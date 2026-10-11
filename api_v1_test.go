package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestV1DeviceAndAnnotationEndpoints(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/devices/bc:32:b2:3b:13:d4", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var device deviceResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &device))
	assert.Equal(t, "bc:32:b2:3b:13:d4", device.MAC)
	assert.Equal(t, "Adam-s-Phone", device.Observed.Hostname)
	assert.NotEmpty(t, device.ActiveLeases)
	assert.Nil(t, device.Annotation)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPatch, "/api/v1/devices/BC-32-B2-3B-13-D4/annotation", bytes.NewBufferString(`{"name":" Adam's phone ","tags":["trusted","mobile"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/devices/bc:32:b2:3b:13:d4", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &device))
	if assert.NotNil(t, device.Annotation) {
		assert.Equal(t, "Adam's phone", device.Annotation.Name)
		assert.Equal(t, []string{"trusted", "mobile"}, device.Annotation.Tags)
	}
}

func TestV1AnnotationRejectsEmptyPatchAndCanRetainDevice(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/devices/aa:bb:cc:dd:ee:ff/annotation", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPatch, "/api/v1/devices/aa:bb:cc:dd:ee:ff/annotation", bytes.NewBufferString(`{"owner":"Operations"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/devices/aa:bb:cc:dd:ee:ff", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var device deviceResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &device))
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", device.MAC)
	assert.NotNil(t, device.Annotation)
}

func TestV1AlertUnacknowledgedFilter(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/alerts?unacknowledged=1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", w.Body.String())
}

func TestV1AlertAcknowledgementRequiresTrue(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/alerts/1", bytes.NewBufferString(`{"acknowledged":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestV1AnnotationPatchPreservesAndClearsFields(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/devices/bc:32:b2:3b:13:d4/annotation", bytes.NewBufferString(`{"name":"Phone","owner":"Operations","notes":"Keep","tags":["trusted"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPatch, "/api/v1/devices/bc:32:b2:3b:13:d4/annotation", bytes.NewBufferString(`{"name":"   ","tags":[]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var annotation annotationResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &annotation))
	assert.Equal(t, "", annotation.Name)
	assert.Equal(t, "Operations", annotation.Owner)
	assert.Equal(t, "Keep", annotation.Notes)
	assert.Empty(t, annotation.Tags)
}

func TestV1AnnotationRejectsInvalidOrDuplicateTags(t *testing.T) {
	router := setupRouter()

	for _, input := range []string{
		`{"tags":[""]}`,
		`{"tags":["trusted"," trusted "]}`,
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/devices/bc:32:b2:3b:13:d4/annotation", bytes.NewBufferString(input))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, input)
	}
}

func TestV1AnnotationDeletion(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/devices/bc:32:b2:3b:13:d4/annotation", bytes.NewBufferString(`{"name":"Phone"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/devices/bc:32:b2:3b:13:d4/annotation", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/devices/bc:32:b2:3b:13:d4/annotation", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestV1DeviceEndpointsRejectInvalidMAC(t *testing.T) {
	router := setupRouter()

	for _, endpoint := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/devices/not-a-mac"},
		{http.MethodGet, "/api/v1/devices/not-a-mac/addresses"},
		{http.MethodPatch, "/api/v1/devices/not-a-mac/annotation"},
	} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(endpoint.method, endpoint.path, bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code, endpoint.path)
	}
}

func TestV1AlertsRetrieveFilterAndAcknowledgeIdempotently(t *testing.T) {
	router, db := setupRouterWithDB()
	acknowledged := time.Date(2024, time.September, 3, 12, 0, 0, 0, time.UTC)
	first := Alert{MAC: "bc:32:b2:3b:13:d4", Hostname: "Phone", IPv4: "192.168.1.9", DetectedAt: time.Date(2024, time.September, 3, 10, 0, 0, 0, time.UTC)}
	second := Alert{MAC: "44:4f:8e:ce:fa:64", Hostname: "wiz_cefa64", IPv4: "192.168.1.143", DetectedAt: time.Date(2024, time.September, 3, 11, 0, 0, 0, time.UTC), Acknowledged: &acknowledged}
	assert.NoError(t, db.Create(&first).Error)
	assert.NoError(t, db.Create(&second).Error)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/alerts?unacknowledged=1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var alerts []Alert
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &alerts))
	if assert.Len(t, alerts, 1) {
		assert.Equal(t, first.ID, alerts[0].ID)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/alerts/999", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPatch, "/api/v1/alerts/1", bytes.NewBufferString(`{"acknowledged":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var acknowledgedAlert Alert
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &acknowledgedAlert))
	if assert.NotNil(t, acknowledgedAlert.Acknowledged) {
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPatch, "/api/v1/alerts/1", bytes.NewBufferString(`{"acknowledged":true}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var repeated Alert
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &repeated))
		assert.Equal(t, *acknowledgedAlert.Acknowledged, *repeated.Acknowledged)
	}
}
