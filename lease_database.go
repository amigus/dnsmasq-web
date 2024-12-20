package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"gorm.io/gorm"
)

type Request struct {
	Received         string `json:"received"`
	Argument         string `json:"argument"`
	Mac              string `json:"mac"`
	IPv4             string `json:"ipv4"`
	ClientID         string `json:"client_id"`
	RequestedOptions string `json:"requested_options"`
}

type Lease struct {
	Mac     string `json:"mac"`
	IPv4    string `json:"ipv4"`
	Added   string `json:"added"`
	Renewed string `json:"renewed"`
}

type Client struct {
	Mac         string `json:"mac"`
	Hostname    string `json:"hostname"`
	ClientID    string `json:"client_id"`
	VendorClass string `json:"vendor_class"`
	Updated     string `json:"updated"`
}

// ipListFromExpression returns the list of IP addresses ipaddr derives from the expression.
func ipListFromExpression(expression string) ([]string, error) {
	ipRange, err := ipaddr.NewIPAddressString(expression).ToAddress()
	if err != nil {
		return nil, err
	}

	var ipList []string
	for ip := ipRange.Iterator(); ip.HasNext(); {
		ipList = append(ipList, ip.Next().WithoutPrefixLen().String())
	}
	return ipList, nil
}

func whereMac(a func(string) string, c *gin.Context, db *gorm.DB, name, field string, required bool) (*gorm.DB, bool) {
	mac := ipaddr.NewMACAddressString(a(name))
	if mac.IsEmpty() {
		if required {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s address is required", name)})
		}
	} else {
		if mac.IsValid() {
			return db.Where(fmt.Sprintf("%s = ?", field), mac.GetAddress().ToColonDelimitedString()), true
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mac address"})
		}
	}
	return nil, false
}

func whereIPv4(a func(string) string, c *gin.Context, db *gorm.DB, name string, required bool) (*gorm.DB, bool) {
	ipv4 := ipaddr.NewIPAddressString(a(name))
	if ipv4.IsEmpty() {
		if required {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s address is required", name)})
		}
	} else {
		if ipv4.IsValid() {
			if ip, err := ipv4.ToAddress(); err == nil {
				return db.Where("ipv4 = ?", ip.WithoutPrefixLen().ToNormalizedString()), true
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ipv4 address"})
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ipv4 address"})
		}
	}
	return nil, false
}

func whereSince(c *gin.Context, db *gorm.DB, oldest time.Time, name, field string, required bool) (*gorm.DB, bool) {
	dateStr := c.Query(name)
	if dateStr != "" {
		if since, err := time.Parse("2006-01-02", dateStr); err == nil {
			if oldest.IsZero() || oldest.Before(since) {
				return db.Where(fmt.Sprintf("%s >= ?", field), since.Format("2006-01-02")), true
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("%s is on or before %s", name, oldest.Format("2006-01-02")),
				})
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		}
	} else if required {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s date is required", name)})
	} else {
		return db, true
	}
	return nil, false
}

func LeaseDatabase(r *gin.Engine, db *gorm.DB) *gin.Engine {
	db.AutoMigrate(&Request{}, &Lease{}, &Client{})

	r.GET("/leases", func(c *gin.Context) {
		var active []struct {
			Mac         string `json:"mac"`
			Hostname    string `json:"hostname"`
			ClientID    string `json:"client_id"`
			VendorClass string `json:"vendor_class"`
			IPv4        string `json:"ipv4"`
			Added       string `json:"added"`
			Renewed     string `json:"renewed"`
		}
		db.Table("clients as c").
			Select("c.mac, c.hostname, c.client_id, c.vendor_class, leases.*").
			Joins("right join leases on c.mac = leases.mac").
			Scan(&active)

		c.JSON(http.StatusOK, active)
	})

	r.GET("/addresses/:mac", func(c *gin.Context) {
		var requests []struct {
			Received         string `json:"received"`
			IPv4             string `json:"ipv4"`
			RequestedOptions string `json:"requested_options"`
			Hostname         string `json:"hostname"`
			VendorClass      string `json:"vendor_class"`
		}
		var ok bool

		query := db.Table("requests as r")
		if query, ok = whereMac(c.Param, c, query, "mac", "r.mac", true); !ok {
			return
		}
		query.Select("r.received, r.ipv4, r.requested_options, c.hostname, c.vendor_class").
			Joins("RIGHT JOIN clients as c ON r.mac = c.mac").
			Order("r.received").
			Scan(&requests)

		type IPHistory struct {
			IPv4             string `json:"ipv4"`
			FirstSeen        string `json:"first_seen"`
			LastSeen         string `json:"last_seen"`
			RequestedOptions string `json:"requested_options"`
			Hostname         string `json:"hostname"`
			VendorClass      string `json:"vendor_class"`
		}

		ipHistory := make(map[string]*IPHistory)
		for _, request := range requests {
			if history, exists := ipHistory[request.IPv4]; exists {
				history.LastSeen = request.Received
			} else {
				ipHistory[request.IPv4] = &IPHistory{
					IPv4:             request.IPv4,
					FirstSeen:        request.Received,
					LastSeen:         request.Received,
					RequestedOptions: request.RequestedOptions,
					Hostname:         request.Hostname,
					VendorClass:      request.VendorClass,
				}
			}
		}

		var ipHistoryList []IPHistory
		for _, history := range ipHistory {
			ipHistoryList = append(ipHistoryList, *history)
		}

		c.JSON(http.StatusOK, ipHistoryList)
	})

	r.GET("/devices/:ipv4", func(c *gin.Context) {
		var requests []struct {
			Received         string `json:"received"`
			Mac              string `json:"mac"`
			RequestedOptions string `json:"requested_options"`
			Hostname         string `json:"hostname"`
			VendorClass      string `json:"vendor_class"`
		}
		var ok bool

		query := db.Table("requests as r")
		if query, ok = whereIPv4(c.Param, c, query, "ipv4", true); !ok {
			return
		}
		query.Select("r.received, r.mac, r.requested_options, c.hostname, c.vendor_class").
			Joins("RIGHT JOIN clients as c ON r.mac = c.mac").
			Order("r.received").
			Scan(&requests)

		type MacHistory struct {
			Mac              string `json:"mac"`
			FirstSeen        string `json:"first_seen"`
			LastSeen         string `json:"last_seen"`
			RequestedOptions string `json:"requested_options"`
			Hostname         string `json:"hostname"`
			VendorClass      string `json:"vendor_class"`
		}

		macHistory := make(map[string]*MacHistory)
		for _, request := range requests {
			if history, exists := macHistory[request.Mac]; exists {
				history.LastSeen = request.Received
			} else {
				macHistory[request.Mac] = &MacHistory{
					Mac:              request.Mac,
					FirstSeen:        request.Received,
					LastSeen:         request.Received,
					RequestedOptions: request.RequestedOptions,
					Hostname:         request.Hostname,
					VendorClass:      request.VendorClass,
				}
			}
		}

		var macHistoryList []MacHistory
		for _, history := range macHistory {
			macHistoryList = append(macHistoryList, *history)
		}

		c.JSON(http.StatusOK, macHistoryList)
	})

	r.GET("/clients", func(c *gin.Context) {
		type ClientRequests struct {
			Client
			Requests int      `json:"requests"`
			IPv4s    []string `json:"ipv4s"`
		}

		subQuery := db.Table("requests").
			Select("mac, GROUP_CONCAT(DISTINCT ipv4 ORDER BY ipv4) as ipv4s").
			Group("mac")

		query := db.Table("requests as r").
			Select("clients.*, COUNT(r.mac) as requests, sub.ipv4s").
			Joins("LEFT JOIN clients ON clients.mac = r.mac").
			Joins("LEFT JOIN (?) as sub ON clients.mac = sub.mac", subQuery).
			Group("r.mac").
			Order("requests")

		var ok bool
		if query, ok = whereSince(c, query, time.Time{}, "since", "r.received", false); !ok {
			return
		}

		var queryResults []struct {
			Client
			Requests int    `json:"requests"`
			IPv4s    string `json:"ipv4s"`
		}
		query.Scan(&queryResults)

		// Convert the comma-separated IPv4s string to a slice of strings
		var results []ClientRequests = make([]ClientRequests, len(queryResults))
		for i, result := range queryResults {
			results[i].Client = result.Client
			results[i].Requests = result.Requests
			if result.IPv4s != "" {
				results[i].IPv4s = strings.Split(result.IPv4s, ",")
			}
		}

		c.JSON(http.StatusOK, results)
	})

	r.GET("/requests", func(c *gin.Context) {
		cidr := c.Query("cidr")
		ipRange := c.Query("range")

		if cidr == "" && ipRange == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Either 'cidr' or 'range' is required",
			})
			return
		}

		var err error // to avoid a type error in ipListFromExpression
		var ipStrings []string

		if cidr != "" {
			ipStrings, err = ipListFromExpression(cidr)
		} else if ipRange != "" {
			ipStrings, err = ipListFromExpression(ipRange)
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var lastIPs []struct {
			IPv4             string `json:"ipv4"`
			Mac              string `json:"mac"`
			Hostname         string `json:"hostname"`
			VendorClass      string `json:"vendor_class"`
			RequestedOptions string `json:"requested_options"`
			Requested        string `json:"requested"`
		}

		// Query the database for the IP addresses
		query := db.Table("requests as r").
			Select("r.ipv4, r.mac, c.hostname, c.vendor_class, r.requested_options,MAX(r.received) as requested").
			Joins("JOIN clients as c ON r.mac = c.mac").
			Where("r.ipv4 IN ?", ipStrings).
			Group("r.ipv4, r.mac, c.hostname, c.vendor_class, r.requested_options").
			Order("requested")

		var ok bool
		if query, ok = whereSince(c, query, time.Time{}, "since", "r.received", false); !ok {
			return
		}
		query.Debug().Scan(&lastIPs)

		type groupedResult struct {
			Mac              string `json:"mac"`
			Hostname         string `json:"hostname"`
			VendorClass      string `json:"vendor_class"`
			RequestedOptions string `json:"requested_options"`
			Requested        string `json:"requested"`
		}

		// Group the results by IPv4
		groupedResults := make(map[string][]groupedResult)

		for _, entry := range lastIPs {
			groupedResults[entry.IPv4] = append(
				groupedResults[entry.IPv4],
				groupedResult{
					Mac:              entry.Mac,
					Hostname:         entry.Hostname,
					VendorClass:      entry.VendorClass,
					RequestedOptions: entry.RequestedOptions,
					Requested:        entry.Requested,
				},
			)
		}

		c.JSON(http.StatusOK, groupedResults)
	})

	return r
}
