package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DeviceAnnotation contains application-owned metadata for a device.
type DeviceAnnotation struct {
	MAC       string `gorm:"primaryKey" json:"mac"`
	Name      string `json:"name"`
	Owner     string `json:"owner"`
	Notes     string `json:"notes"`
	Tags      string `json:"-"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Alert is a persistent event detected for a device.
type Alert struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	MAC          string     `json:"mac"`
	Hostname     string     `json:"hostname"`
	IPv4         string     `json:"ipv4"`
	DetectedAt   time.Time  `json:"detected_at"`
	Acknowledged *time.Time `json:"acknowledged_at"`
}

type annotationResponse struct {
	Name  string   `json:"name"`
	Owner string   `json:"owner"`
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`
}

type observedDeviceResponse struct {
	Hostname     string `json:"hostname"`
	ClientID     string `json:"client_id"`
	VendorClass  string `json:"vendor_class"`
	FirstSeen    string `json:"first_seen"`
	LastSeen     string `json:"last_seen"`
	RequestCount int    `json:"request_count"`
}

type leaseResponse struct {
	IPv4    string `json:"ipv4"`
	Added   string `json:"added"`
	Renewed string `json:"renewed"`
}

type deviceResponse struct {
	MAC          string                 `json:"mac"`
	Observed     observedDeviceResponse `json:"observed"`
	ActiveLeases []leaseResponse        `json:"active_leases"`
	Annotation   *annotationResponse    `json:"annotation,omitempty"`
}

type annotationInput struct {
	Name  *string   `json:"name"`
	Owner *string   `json:"owner"`
	Notes *string   `json:"notes"`
	Tags  *[]string `json:"tags"`
}

func annotationFor(annotation DeviceAnnotation) *annotationResponse {
	tags := []string{}
	if annotation.Tags != "" {
		if err := json.Unmarshal([]byte(annotation.Tags), &tags); err != nil {
			return nil
		}
	}
	return &annotationResponse{
		Name:  annotation.Name,
		Owner: annotation.Owner,
		Notes: annotation.Notes,
		Tags:  tags,
	}
}

func devices(db *gorm.DB) ([]deviceResponse, error) {
	type deviceRow struct {
		MAC          string
		Hostname     string
		ClientID     string
		VendorClass  string
		FirstSeen    string
		LastSeen     string
		RequestCount int
	}

	var rows []deviceRow
	if err := db.Table("requests as r").
		Select("r.mac, MAX(c.hostname) as hostname, MAX(c.client_id) as client_id, MAX(c.vendor_class) as vendor_class, MIN(r.received) as first_seen, MAX(r.received) as last_seen, COUNT(*) as request_count").
		Joins("LEFT JOIN clients as c ON c.mac = r.mac").
		Group("r.mac").
		Order("r.mac").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	var annotations []DeviceAnnotation
	if err := db.Find(&annotations).Error; err != nil {
		return nil, err
	}
	byMAC := make(map[string]DeviceAnnotation, len(annotations))
	for _, annotation := range annotations {
		byMAC[annotation.MAC] = annotation
	}

	var leases []Lease
	if err := db.Find(&leases).Error; err != nil {
		return nil, err
	}
	leasesByMAC := make(map[string][]leaseResponse)
	for _, lease := range leases {
		leasesByMAC[lease.Mac] = append(leasesByMAC[lease.Mac], leaseResponse{IPv4: lease.IPv4, Added: lease.Added, Renewed: lease.Renewed})
	}

	result := make([]deviceResponse, 0, len(rows)+len(annotations))
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		device := deviceResponse{
			MAC: row.MAC,
			Observed: observedDeviceResponse{
				Hostname: row.Hostname, ClientID: row.ClientID, VendorClass: row.VendorClass,
				FirstSeen: row.FirstSeen, LastSeen: row.LastSeen, RequestCount: row.RequestCount,
			},
			ActiveLeases: append([]leaseResponse{}, leasesByMAC[row.MAC]...),
		}
		if annotation, ok := byMAC[row.MAC]; ok {
			device.Annotation = annotationFor(annotation)
		}
		result = append(result, device)
		seen[row.MAC] = true
	}
	for _, annotation := range annotations {
		if !seen[annotation.MAC] {
			result = append(result, deviceResponse{MAC: annotation.MAC, Annotation: annotationFor(annotation)})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].MAC < result[j].MAC })
	return result, nil
}

func validateAnnotation(input annotationInput) error {
	if input.Name != nil && len(strings.TrimSpace(*input.Name)) > 256 {
		return fmt.Errorf("name must be at most 256 bytes")
	}
	if input.Owner != nil && len(strings.TrimSpace(*input.Owner)) > 256 {
		return fmt.Errorf("owner must be at most 256 bytes")
	}
	if input.Notes != nil && len(strings.TrimSpace(*input.Notes)) > 4096 {
		return fmt.Errorf("notes must be at most 4096 bytes")
	}
	if input.Tags != nil {
		if len(*input.Tags) > 32 {
			return fmt.Errorf("at most 32 tags are allowed")
		}
		seen := make(map[string]bool, len(*input.Tags))
		for _, tag := range *input.Tags {
			tag = strings.TrimSpace(tag)
			if len(tag) == 0 || len(tag) > 64 {
				return fmt.Errorf("tags must be 1 to 64 bytes")
			}
			if seen[tag] {
				return fmt.Errorf("tags must be unique")
			}
			seen[tag] = true
		}
	}
	return nil
}

func normalizedMAC(c *gin.Context) (string, bool) {
	mac, err := validateMAC(c.Param("mac"))
	if err != nil || !mac.ToAddressString().IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		return "", false
	}
	return mac.ToColonDelimitedString(), true
}

// ApiV1 registers the versioned canonical device and alert API. The existing
// unversioned routes remain compatibility endpoints.
func ApiV1(r *gin.Engine, db *gorm.DB) {
	db.AutoMigrate(&DeviceAnnotation{}, &Alert{})
	api := r.Group("/api/v1")

	api.GET("/devices", func(c *gin.Context) {
		result, err := devices(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	api.GET("/devices/:mac", func(c *gin.Context) {
		mac, ok := normalizedMAC(c)
		if !ok {
			return
		}
		result, err := devices(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, device := range result {
			if device.MAC == mac {
				c.JSON(http.StatusOK, device)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "no such device"})
	})

	api.GET("/devices/:mac/addresses", func(c *gin.Context) {
		mac, ok := normalizedMAC(c)
		if !ok {
			return
		}
		var result []struct {
			IPv4      string `json:"ipv4"`
			FirstSeen string `json:"first_seen"`
			LastSeen  string `json:"last_seen"`
		}
		if err := db.Table("requests").Select("ipv4, MIN(received) as first_seen, MAX(received) as last_seen").Where("mac = ?", mac).Group("ipv4").Order("ipv4").Scan(&result).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	api.PATCH("/devices/:mac/annotation", func(c *gin.Context) {
		mac, ok := normalizedMAC(c)
		if !ok {
			return
		}
		var input annotationInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if input.Name == nil && input.Owner == nil && input.Notes == nil && input.Tags == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one annotation field is required"})
			return
		}
		if err := validateAnnotation(input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var annotation DeviceAnnotation
		err := db.First(&annotation, "mac = ?", mac).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		annotation.MAC = mac
		if input.Name != nil {
			annotation.Name = strings.TrimSpace(*input.Name)
		}
		if input.Owner != nil {
			annotation.Owner = strings.TrimSpace(*input.Owner)
		}
		if input.Notes != nil {
			annotation.Notes = strings.TrimSpace(*input.Notes)
		}
		if input.Tags != nil {
			for i := range *input.Tags {
				(*input.Tags)[i] = strings.TrimSpace((*input.Tags)[i])
			}
			tags, err := json.Marshal(*input.Tags)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			annotation.Tags = string(tags)
		}
		if err := db.Save(&annotation).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, annotationFor(annotation))
	})

	api.DELETE("/devices/:mac/annotation", func(c *gin.Context) {
		mac, ok := normalizedMAC(c)
		if !ok {
			return
		}
		result := db.Delete(&DeviceAnnotation{}, "mac = ?", mac)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "no annotation"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	api.GET("/alerts", func(c *gin.Context) {
		query := db.Order("detected_at DESC")
		if c.Query("unacknowledged") == "1" {
			query = query.Where("acknowledged IS NULL")
		}
		var alerts []Alert
		if err := query.Find(&alerts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, alerts)
	})

	api.GET("/alerts/:id", func(c *gin.Context) {
		var alert Alert
		if err := db.First(&alert, c.Param("id")).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "no such alert"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, alert)
	})

	api.PATCH("/alerts/:id", func(c *gin.Context) {
		var input struct {
			Acknowledged *bool `json:"acknowledged"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Acknowledged == nil || !*input.Acknowledged {
			c.JSON(http.StatusBadRequest, gin.H{"error": "acknowledged must be true"})
			return
		}
		var alert Alert
		if err := db.First(&alert, c.Param("id")).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "no such alert"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		if alert.Acknowledged == nil {
			now := time.Now().UTC()
			alert.Acknowledged = &now
		}
		if err := db.Save(&alert).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, alert)
	})
}
