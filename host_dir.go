package main

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"

	"github.com/amigus/dnsmasq-web/internal/reservations"
	"github.com/gin-gonic/gin"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"gorm.io/gorm"
)

type reservationData struct {
	Tags      []string `json:"tags"`
	IPv4      string   `json:"ipv4"`
	FromLease bool     `json:"from_lease,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
	LeaseTime string   `json:"lease_time,omitempty"`
}

type reservation struct {
	MAC string `json:"mac" binding:"required"`
	reservationData
}

func validateMAC(mac string) (*ipaddr.MACAddress, error) {
	addr, err := ipaddr.NewMACAddressString(mac).ToAddress()
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func validateIPv4(ipv4 string) (*ipaddr.IPAddress, error) {
	addr, err := ipaddr.NewIPAddressString(ipv4).ToAddress()
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func createReservationFile(input reservation, c *gin.Context, db *gorm.DB, hostDir string, overwrite bool) {
	mac, err := validateMAC(input.MAC)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		return
	}

	if (input.IPv4 == "") == !input.FromLease {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exactly one of ipv4 or from_lease is required"})
		return
	}

	if input.FromLease {
		var lease Lease
		if err := db.Where("mac = ?", mac.ToColonDelimitedString()).First(&lease).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no current lease; an explicit IPv4 is required"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}
		input.IPv4 = lease.IPv4
	}

	ipv4, err := validateIPv4(input.IPv4)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid IPv4 address"})
		return
	}

	if existing, err := findReservationByIPv4(ipv4.String(), hostDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else if existing != nil && existing.MAC != mac.ToColonDelimitedString() {
		c.JSON(http.StatusConflict, gin.H{"error": "ipv4 " + ipv4.String() + " is already reserved for " + existing.MAC})
		return
	}

	var leased Lease
	if err := db.Where("ipv4 = ?", ipv4.String()).First(&leased).Error; err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else if err == nil && leased.Mac != mac.ToColonDelimitedString() && c.Query("force") != "1" {
		c.JSON(http.StatusConflict, gin.H{"error": "ipv4 " + ipv4.String() + " is currently leased to " + leased.Mac + "; retry with force=1 to override"})
		return
	}

	fileContent := reservations.Format(reservations.FileRecord{
		MAC:       mac.ToColonDelimitedString(),
		Tags:      input.Tags,
		IPv4:      ipv4.String(),
		Hostname:  input.Hostname,
		LeaseTime: input.LeaseTime,
	})

	filePath := filepath.Join(hostDir, mac.ToNormalizedString())
	if _, err := os.Stat(filePath); err == nil && !overwrite {
		c.JSON(http.StatusConflict, gin.H{"error": "exists"})
		return
	}
	if err := os.WriteFile(filePath, []byte(fileContent), 0640); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "success", "from_lease": input.FromLease})
}

func updateReservationFile(input reservationData, mac string, c *gin.Context, db *gorm.DB, hostDir string) {
	createReservationFile(reservation{MAC: mac, reservationData: input}, c, db, hostDir, true)
}

func deleteReservationFile(c *gin.Context, hostDir string) {
	mac, err := validateMAC(c.Param("mac"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		return
	}
	filePath := filepath.Join(hostDir, mac.ToNormalizedString())
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no such reservation"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func findReservationByIPv4(ipv4, hostDir string) (*reservation, error) {
	var found *reservation
	err := filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		mac, err := validateMAC(info.Name())
		if err != nil {
			return nil
		}
		res, err := readReservationFile(mac.ToNormalizedString(), hostDir)
		if err != nil {
			return err
		}
		if res.IPv4 == ipv4 {
			found = &res
		}
		return nil
	})
	return found, err
}

func readReservationFile(mac string, hostDir string) (reservation, error) {
	if _, err := validateMAC(mac); err != nil {
		return reservation{}, err
	}

	filePath := filepath.Join(hostDir, mac)
	file, err := os.Open(filePath)
	if err != nil {
		return reservation{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		record, err := reservations.Parse(scanner.Text())
		if err != nil {
			return reservation{}, err
		}

		res := reservation{
			MAC: record.MAC,
			reservationData: reservationData{
				Tags:      record.Tags,
				IPv4:      record.IPv4,
				Hostname:  record.Hostname,
				LeaseTime: record.LeaseTime,
			},
		}

		return res, nil
	}

	if err := scanner.Err(); err != nil {
		return reservation{}, err
	}
	return reservation{}, nil
}

func getReservationFile(c *gin.Context, hostDir string) {
	macParam := c.Param("mac")
	if macParam != "" {
		mac, err := validateMAC(macParam)
		if err != nil || !mac.ToAddressString().IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MAC address"})
		} else {
			if res, err := readReservationFile(mac.ToNormalizedString(), hostDir); err == nil {
				c.JSON(http.StatusOK, res)
			} else {
				if os.IsNotExist(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "no such reservation"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				}
			}
		}
	} else {
		var reservations []reservation
		err := filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				macAddr, err := validateMAC(info.Name())
				if err == nil {
					if res, err := readReservationFile(macAddr.ToNormalizedString(), hostDir); err == nil {
						reservations = append(reservations, res)
					} else {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, reservations)
	}
}

func DhcpHostDir(r *gin.Engine, db *gorm.DB, hostDir string) *gin.Engine {
	r.POST("/reservations", func(c *gin.Context) {
		var input struct {
			MAC string `json:"mac" binding:"required"`
			reservationData
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		createReservationFile(input, c, db, hostDir, false)
	})

	r.PUT("/reservations/:mac", func(c *gin.Context) {
		var input reservationData
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updateReservationFile(input, c.Param("mac"), c, db, hostDir)
	})

	r.DELETE("/reservations/:mac", func(c *gin.Context) {
		deleteReservationFile(c, hostDir)
	})

	r.GET("/reservations", func(c *gin.Context) {
		getReservationFile(c, hostDir)
	})

	r.GET("/reservations/:mac", func(c *gin.Context) {
		getReservationFile(c, hostDir)
	})

	return r
}
