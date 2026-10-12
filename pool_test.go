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

func TestPoolEndpoint(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pool", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report poolResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &report))
	assert.Equal(t, "192.168.1.0/24", report.Subnet)
	assert.Equal(t, 254, report.Total)
	assert.Equal(t, 10, report.Used)
	assert.Equal(t, 244, report.Free)
	assert.Equal(t, 3.9, report.Utilization)
}

func TestV1PoolEndpoint(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/pool", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report poolResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &report))
	assert.Equal(t, "192.168.1.0/24", report.Subnet)
	assert.Equal(t, 10, report.Used)
}

func TestPoolEndpointNoSubnet(t *testing.T) {
	r := gin.Default()
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.Exec(testDatabaseSQL)
	router := LeaseDatabase(r, db, "", 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pool", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPoolReportSmallSubnets(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, db.AutoMigrate(&Lease{}))

	report, err := poolReport(db, "192.168.1.0/31")
	assert.NoError(t, err)
	assert.Equal(t, 2, report.Total)
	assert.Equal(t, 0, report.Used)
	assert.Equal(t, 2, report.Free)

	report, err = poolReport(db, "192.168.1.1/32")
	assert.NoError(t, err)
	assert.Equal(t, 1, report.Total)
	assert.Equal(t, 0, report.Used)
	assert.Equal(t, 0.0, report.Utilization)
}
