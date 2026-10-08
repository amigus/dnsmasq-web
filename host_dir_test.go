package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouterForReservationsTests() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	hostDir := "./test_hosts"
	os.Mkdir(hostDir, 0755)
	DhcpHostDir(r, hostDir)
	return r
}

func TestCreateReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100",
		"tags": ["tag1", "tag2"],
		"hostname": "test-host",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.FileExists(t, filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))
}

func TestCreateMinimalReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	filePath := filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e")
	assert.FileExists(t, filePath)
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "00:1a:2b:3c:4d:5e,192.168.1.100", string(content))
}

func TestCreateReservationWithInvalidIPv4(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.256",
		"tags": ["tag1", "tag2"],
		"hostname": "test-host",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateReservationWithMissingMAC(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"ipv4": "192.168.1.100",
		"tags": ["tag1", "tag2"],
		"hostname": "test-host",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateReservationWithInvalidMAC(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5G",
		"ipv4": "192.168.1.101",
		"tags": ["tag3"],
		"hostname": "updated-host",
		"lease_time": "48h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	// First create a reservation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100",
		"tags": ["tag1", "tag2"],
		"hostname": "test-host",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Now update the reservation
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/reservations/00:1A:2B:3C:4D:5E", strings.NewReader(`{
		"ipv4": "192.168.1.101",
		"tags": ["tag3"],
		"hostname": "updated-host",
		"lease_time": "48h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	content, _ := os.ReadFile(filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))
	assert.Contains(t, string(content), "192.168.1.101")
	assert.Contains(t, string(content), "set:tag3")
	assert.Contains(t, string(content), "updated-host")
	assert.Contains(t, string(content), "48h")
}

func TestUpdateReservationNoTags(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	// First create a reservation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100",
		"hostname": "test-host"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	content, _ := os.ReadFile(filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))
	assert.Equal(t, "00:1a:2b:3c:4d:5e,192.168.1.100,test-host", string(content))
}

func TestDeleteReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	// First create a reservation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100",
		"tags": ["tag1", "tag2"],
		"hostname": "test-host",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.FileExists(t, filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))

	// Now delete the reservation
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/reservations/00:1A:2B:3C:4D:5E", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoFileExists(t, filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))
}

func TestDeleteNonexistentReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/reservations/00:1A:2B:3C:4D:5E", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
func TestGetAllReservations(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	// Create a few reservations
	reservations := []string{
		`{"mac": "00:1A:2B:3C:4D:5E", "ipv4": "192.168.1.100", "tags": ["tag1"], "hostname": "host1", "lease_time": "24h"}`,
		`{"mac": "00:1A:2B:3C:4D:5F", "ipv4": "192.168.1.101", "tags": ["tag2"], "hostname": "host2", "lease_time": "24h"}`,
	}

	for _, res := range reservations {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(res))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	// Get all reservations
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/reservations", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "00:1a:2b:3c:4d:5e")
	assert.Contains(t, w.Body.String(), "00:1a:2b:3c:4d:5f")
}

func TestGetReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	// Create a reservation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100",
		"tags": ["tag1"],
		"hostname": "host1",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Get the reservation
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/reservations/00:1A:2B:3C:4D:5E", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "00:1a:2b:3c:4d:5e")
	assert.Contains(t, w.Body.String(), "192.168.1.100")
	assert.Contains(t, w.Body.String(), "host1")
	assert.Contains(t, w.Body.String(), "24h")
}

func TestGetReservationWithMultipleTags(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/reservations", strings.NewReader(`{
		"mac": "00:1A:2B:3C:4D:5E",
		"ipv4": "192.168.1.100",
		"tags": ["tag1", "tag2"],
		"hostname": "host1",
		"lease_time": "24h"
	}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/reservations/00:1A:2B:3C:4D:5E", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "192.168.1.100")
	assert.Contains(t, w.Body.String(), "tag1")
	assert.Contains(t, w.Body.String(), "tag2")
	assert.NotContains(t, w.Body.String(), "set:tag2\",\"ipv4\":\"set:tag2")
}

func TestGetNonexistentReservation(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/reservations/00:1A:2B:3C:4D:5E", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReservationWithInvalidMAC(t *testing.T) {
	r := setupRouterForReservationsTests()
	defer os.RemoveAll("./test_hosts")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/reservations/00:1A:2B:3C:4D:5G", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReadReservationFileRejectsTraversalShapedMAC(t *testing.T) {
	hostDir := "./test_hosts"
	os.Mkdir(hostDir, 0755)
	defer os.RemoveAll(hostDir)

	// A valid reservation inside hostDir keeps the happy path observable.
	validPath := filepath.Join(hostDir, "00:1a:2b:3c:4d:5e")
	assert.NoError(t, os.WriteFile(validPath, []byte("00:1a:2b:3c:4d:5e,192.168.1.100"), 0640))

	// A sentinel outside hostDir that a traversal would otherwise reach.
	sentinel := "./test_traversal_target"
	assert.NoError(t, os.WriteFile(sentinel, []byte("00:1a:2b:3c:4d:5e,10.0.0.1"), 0640))
	defer os.Remove(sentinel)

	cases := []struct {
		name  string
		input string
	}{
		{"parent dir", ".."},
		{"parent traversal", "../.."},
		{"embedded traversal", "a/../b"},
		{"escape to sentinel", "../test_traversal_target"},
		{"backslash traversal", `..\..\etc`},
		{"encoded traversal", "..%2f..%2f"},
		{"traversal after valid mac", "00:1a:2b:3c:4d:5e/../../test_traversal_target"},
		{"nul byte", "00:1a:2b:3c:4d:5e\x00"},
		{"absolute path", "/etc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readReservationFile(tc.input, hostDir)
			assert.Error(t, err)
		})
	}

	// Positive control: a normalized MAC still reads successfully.
	res, err := readReservationFile("00:1a:2b:3c:4d:5e", hostDir)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.100", res.IPv4)
}
