package main

import (
	"embed"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//go:embed dashboard_templates/*.html
var dashboardTemplates embed.FS

// dashboardStat is one overview counter on the dashboard page.
type dashboardStat struct {
	Label string
	Value string
}

// dashboardSection is one table section of the dashboard page.
type dashboardSection struct {
	Title   string
	Note    string
	Headers []string
	Rows    [][]string
}

// dashboardData is the template payload for the dashboard page.
type dashboardData struct {
	Pool      *poolResponse
	Stats     []dashboardStat
	Sections  []dashboardSection
	CanLogout bool
}

// Dashboard registers the read-only server-rendered dashboard plus its login and
// logout routes. Call it after the auth middleware and before the data routes so
// gin freezes the middleware into every route.
func Dashboard(r *gin.Engine, db *gorm.DB, subnet, hostDir string, ttc TokenChecker) *gin.Engine {
	tmpl := template.Must(template.ParseFS(dashboardTemplates, "dashboard_templates/*.html"))
	r.SetHTMLTemplate(tmpl)

	// Without token checking (foreground mode) there is nothing to log into.
	if ttc == nil {
		return r
	}

	r.GET(dashboardLoginPath, func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})

	// Login validates a token issued through the Unix socket without consuming
	// one of its uses, then carries it in a browser-session cookie.
	r.POST(dashboardLoginPath, func(c *gin.Context) {
		c.SetSameSite(http.SameSiteStrictMode)
		token := c.PostForm("token")
		if !ttc.Validate(token) {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{"Error": "Invalid token"})
			return
		}
		c.SetCookie(dashboardCookieName, token, 0, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/")
	})

	// Logout only clears the caller's own cookie, so it stays exempt from auth.
	r.POST(dashboardLogoutPath, func(c *gin.Context) {
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(dashboardCookieName, "", -1, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, dashboardLoginPath)
	})

	r.GET("/", func(c *gin.Context) {
		data, err := collectDashboardData(db, subnet, hostDir)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"Error": err.Error()})
			return
		}
		data.CanLogout = true
		c.HTML(http.StatusOK, "dashboard.html", data)
	})

	return r
}

// collectDashboardData gathers every section shown on the dashboard page.
func collectDashboardData(db *gorm.DB, subnet, hostDir string) (dashboardData, error) {
	data := dashboardData{}

	if subnet != "" {
		report, err := poolReport(db, subnet)
		if err != nil {
			return data, err
		}
		data.Pool = &report
	}

	deviceList, err := devices(db)
	if err != nil {
		return data, err
	}

	var alerts []Alert
	if err := db.Where("acknowledged IS NULL").Order("detected_at DESC").Find(&alerts).Error; err != nil {
		return data, err
	}

	var leases []Lease
	if err := db.Find(&leases).Error; err != nil {
		return data, err
	}

	var clients []Client
	if err := db.Find(&clients).Error; err != nil {
		return data, err
	}

	var reservations []reservation
	reservationNote := "not configured; start the server with -h"
	if hostDir != "" {
		reservations, err = listReservations(hostDir)
		if err != nil {
			return data, err
		}
		reservationNote = ""
	}

	data.Stats = []dashboardStat{
		{Label: "Leases", Value: strconv.Itoa(len(leases))},
		{Label: "Clients", Value: strconv.Itoa(len(clients))},
		{Label: "Devices", Value: strconv.Itoa(len(deviceList))},
		{Label: "Unacknowledged alerts", Value: strconv.Itoa(len(alerts))},
		{Label: "Reservations", Value: strconv.Itoa(len(reservations))},
	}

	leaseSection := dashboardSection{Title: "Leases", Headers: []string{"MAC", "IPv4", "Added", "Renewed"}}
	for _, lease := range leases {
		leaseSection.Rows = append(leaseSection.Rows, []string{lease.Mac, lease.IPv4, lease.Added, lease.Renewed})
	}

	clientSection := dashboardSection{Title: "Clients", Headers: []string{"MAC", "Hostname", "Client ID", "Vendor class", "Updated"}}
	for _, client := range clients {
		clientSection.Rows = append(clientSection.Rows, []string{client.Mac, client.Hostname, client.ClientID, client.VendorClass, client.Updated})
	}

	deviceSection := dashboardSection{Title: "Devices", Headers: []string{"MAC", "Name", "Owner", "IPv4", "Last seen"}}
	for _, device := range deviceList {
		name := device.Observed.Hostname
		owner := ""
		if device.Annotation != nil {
			if device.Annotation.Name != "" {
				name = device.Annotation.Name
			}
			owner = device.Annotation.Owner
		}
		ipv4 := ""
		if len(device.ActiveLeases) > 0 {
			ipv4 = device.ActiveLeases[0].IPv4
		}
		deviceSection.Rows = append(deviceSection.Rows, []string{device.MAC, name, owner, ipv4, device.Observed.LastSeen})
	}

	alertSection := dashboardSection{Title: "Unacknowledged alerts", Headers: []string{"MAC", "Hostname", "IPv4", "Detected"}}
	for _, alert := range alerts {
		alertSection.Rows = append(alertSection.Rows, []string{alert.MAC, alert.Hostname, alert.IPv4, alert.DetectedAt.Format("2006-01-02 15:04")})
	}

	reservationSection := dashboardSection{Title: "Reservations", Note: reservationNote, Headers: []string{"MAC", "IPv4", "Hostname", "Lease time"}}
	for _, res := range reservations {
		reservationSection.Rows = append(reservationSection.Rows, []string{res.MAC, res.IPv4, res.Hostname, res.LeaseTime})
	}

	data.Sections = []dashboardSection{leaseSection, clientSection, deviceSection, alertSection, reservationSection}
	return data, nil
}

// listReservations lists every reservation file in the host directory.
func listReservations(hostDir string) ([]reservation, error) {
	var found []reservation
	err := filepath.Walk(hostDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		macAddr, err := validateMAC(info.Name())
		if err != nil {
			return nil
		}
		res, err := readReservationFile(macAddr.ToNormalizedString(), hostDir)
		if err != nil {
			return err
		}
		found = append(found, res)
		return nil
	})
	return found, err
}
