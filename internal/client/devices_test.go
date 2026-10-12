package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient builds a no-auth client pointed at an httptest server.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := &Config{Server: server.Listener.Addr().String(), NoAuth: true}
	cl, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return cl
}

func TestGetV1Devices(t *testing.T) {
	var path, method string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"mac":"bc:32:b2:3b:13:d4","observed":{"hostname":"phone","request_count":3},"active_leases":[{"ipv4":"192.168.1.9"}],"annotation":{"name":"Phone","owner":"Adam","notes":"","tags":["trusted"]}}]`))
	})

	devices, err := cl.GetV1Devices()
	if err != nil {
		t.Fatalf("GetV1Devices: %v", err)
	}
	if method != http.MethodGet || path != "/api/v1/devices" {
		t.Fatalf("expected GET /api/v1/devices, got %s %s", method, path)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	device := devices[0]
	if device.MAC != "bc:32:b2:3b:13:d4" {
		t.Fatalf("unexpected mac %q", device.MAC)
	}
	if device.Observed.Hostname != "phone" || device.Observed.RequestCount != 3 {
		t.Fatalf("unexpected observation %+v", device.Observed)
	}
	if len(device.ActiveLeases) != 1 || device.ActiveLeases[0].IPv4 != "192.168.1.9" {
		t.Fatalf("unexpected leases %+v", device.ActiveLeases)
	}
	if device.Annotation == nil || device.Annotation.Name != "Phone" || len(device.Annotation.Tags) != 1 || device.Annotation.Tags[0] != "trusted" {
		t.Fatalf("unexpected annotation %+v", device.Annotation)
	}
}

func TestGetV1DeviceNotFound(t *testing.T) {
	var path string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no such device"}`))
	})

	_, err := cl.GetV1Device("bc:32:b2:3b:13:d4")
	if err == nil || !strings.Contains(err.Error(), "no such device") {
		t.Fatalf("expected no-such-device error, got %v", err)
	}
	if path != "/api/v1/devices/bc:32:b2:3b:13:d4" {
		t.Fatalf("unexpected path %q", path)
	}
}

func TestGetV1DeviceAddresses(t *testing.T) {
	var path string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`[{"ipv4":"192.168.1.9","first_seen":"2024-09-03 10:07:21","last_seen":"2024-09-03 12:37:22"}]`))
	})

	addresses, err := cl.GetV1DeviceAddresses("bc:32:b2:3b:13:d4")
	if err != nil {
		t.Fatalf("GetV1DeviceAddresses: %v", err)
	}
	if path != "/api/v1/devices/bc:32:b2:3b:13:d4/addresses" {
		t.Fatalf("unexpected path %q", path)
	}
	if len(addresses) != 1 || addresses[0].IPv4 != "192.168.1.9" || addresses[0].FirstSeen != "2024-09-03 10:07:21" {
		t.Fatalf("unexpected addresses %+v", addresses)
	}
}

func TestUpdateV1DeviceAnnotation(t *testing.T) {
	var method, path string
	var body []byte
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"name":"Living-room TV","owner":"","notes":"","tags":[]}`))
	})

	name := "Living-room TV"
	tags := []string{}
	annotation, err := cl.UpdateV1DeviceAnnotation("bc:32:b2:3b:13:d4", AnnotationPatch{Name: &name, Tags: &tags})
	if err != nil {
		t.Fatalf("UpdateV1DeviceAnnotation: %v", err)
	}
	if method != http.MethodPatch || path != "/api/v1/devices/bc:32:b2:3b:13:d4/annotation" {
		t.Fatalf("expected PATCH annotation endpoint, got %s %s", method, path)
	}

	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("sent body is not JSON: %v", err)
	}
	if sent["name"] != "Living-room TV" {
		t.Fatalf("unexpected name %v", sent["name"])
	}
	if _, ok := sent["owner"]; ok {
		t.Fatalf("omitted owner must not be sent")
	}
	if _, ok := sent["notes"]; ok {
		t.Fatalf("omitted notes must not be sent")
	}
	sentTags, ok := sent["tags"].([]any)
	if !ok || len(sentTags) != 0 {
		t.Fatalf("clearing tags must send an empty array, got %v", sent["tags"])
	}
	if annotation == nil || annotation.Name != "Living-room TV" || len(annotation.Tags) != 0 {
		t.Fatalf("unexpected annotation %+v", annotation)
	}
}

func TestDeleteV1DeviceAnnotation(t *testing.T) {
	var method, path string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	if err := cl.DeleteV1DeviceAnnotation("bc:32:b2:3b:13:d4"); err != nil {
		t.Fatalf("DeleteV1DeviceAnnotation: %v", err)
	}
	if method != http.MethodDelete || path != "/api/v1/devices/bc:32:b2:3b:13:d4/annotation" {
		t.Fatalf("expected DELETE annotation endpoint, got %s %s", method, path)
	}
}

func TestDeleteV1DeviceAnnotationNotFound(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no annotation"}`))
	})

	err := cl.DeleteV1DeviceAnnotation("bc:32:b2:3b:13:d4")
	if err == nil || !strings.Contains(err.Error(), "no annotation") {
		t.Fatalf("expected no-annotation error, got %v", err)
	}
}
