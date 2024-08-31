package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouter() *gin.Engine {
	r := gin.Default()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	db.Exec(testDatabaseSQL)

	return LeaseDatabase(r, db, 30, "192.168.1.0/24")
}

func TestLeasesEndpoint(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/leases", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []struct {
		Mac         string `json:"mac"`
		Hostname    string `json:"hostname"`
		ClientID    string `json:"client_id"`
		VendorClass string `json:"vendor_class"`
		IPv4        string `json:"ipv4"`
		Added       string `json:"added"`
		Renewed     string `json:"renewed"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Equal(t, "44:4f:8e:ce:fa:64", response[0].Mac)
	assert.Equal(t, "192.168.1.143", response[0].IPv4)
	assert.Equal(t, "2024-09-03 12:57:54", response[0].Renewed)
}

func TestAddressesEndpoint(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/addresses/bc:32:b2:3b:13:d4", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []struct {
		IPv4             string `json:"ipv4"`
		FirstSeen        string `json:"first_seen"`
		LastSeen         string `json:"last_seen"`
		RequestedOptions string `json:"requested_options"`
		Hostname         string `json:"hostname"`
		VendorClass      string `json:"vendor_class"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Equal(t, "2024-09-03 10:07:21", response[0].FirstSeen)
	assert.Equal(t, "192.168.1.9", response[0].IPv4)
	assert.Equal(t, "Adam-s-Phone", response[0].Hostname)
}

func TestDevicesEndpoint(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/devices/192.168.1.108", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []struct {
		Mac       string `json:"mac"`
		Hostname  string `json:"hostname"`
		ClientID  string `json:"client_id"`
		FirstSeen string `json:"first_seen"`
		LastSeen  string `json:"last_seen"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Equal(t, "2024-09-03 10:25:07", response[0].FirstSeen)
	assert.Equal(t, "wiz_ca8fe0", response[0].Hostname)
}

func TestClientsEndpoint(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/clients?since=2024-09-01", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []struct {
		Client
		Requests int      `json:"requests"`
		IPv4s    []string `json:"ipv4s"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Equal(t, "192.168.1.107", response[8].IPv4s[0])
	assert.Equal(t, "wiz_ca8fe0", response[9].Hostname)
}

func TestRequestsEndpointCIDR(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/requests?cidr=192.168.1.0/28", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string][]struct {
		Mac              string `json:"mac"`
		Hostname         string `json:"hostname"`
		VendorClass      string `json:"vendor_class"`
		RequestedOptions string `json:"requested_options"`
		Requested        string `json:"requested"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "192.168.1.9")
	assert.Equal(t, "Adam-s-Phone", response["192.168.1.9"][0].Hostname)
	assert.Equal(t, "android-dhcp-14", response["192.168.1.9"][0].VendorClass)
}

func TestRequestsEndpointRange(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/requests?range=192.168.1.1-15", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string][]struct {
		Mac              string `json:"mac"`
		Hostname         string `json:"hostname"`
		VendorClass      string `json:"vendor_class"`
		RequestedOptions string `json:"requested_options"`
		Requested        string `json:"requested"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "192.168.1.9")
	assert.Equal(t, "Adam-s-Phone", response["192.168.1.9"][0].Hostname)
}
