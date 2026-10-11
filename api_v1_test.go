package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
