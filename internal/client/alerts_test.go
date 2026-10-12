package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetV1Alerts(t *testing.T) {
	var paths []string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		_, _ = w.Write([]byte(`[{"id":7,"mac":"aa:bb:cc:dd:ee:ff","hostname":"tv","ipv4":"192.168.1.50","detected_at":"2026-10-12T12:00:00Z","acknowledged_at":null}]`))
	})

	alerts, err := cl.GetV1Alerts(true)
	if err != nil {
		t.Fatalf("GetV1Alerts: %v", err)
	}
	if len(alerts) != 1 || alerts[0].ID != 7 || alerts[0].MAC != "aa:bb:cc:dd:ee:ff" || alerts[0].Hostname != "tv" {
		t.Fatalf("unexpected alerts %+v", alerts)
	}
	if alerts[0].Acknowledged != nil {
		t.Fatalf("expected nil acknowledgement, got %v", *alerts[0].Acknowledged)
	}

	if _, err := cl.GetV1Alerts(false); err != nil {
		t.Fatalf("GetV1Alerts: %v", err)
	}

	if len(paths) != 2 || paths[0] != "/api/v1/alerts?unacknowledged=1" || paths[1] != "/api/v1/alerts?" {
		t.Fatalf("unexpected request paths %v", paths)
	}
}

func TestGetV1Alert(t *testing.T) {
	var path string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"id":7,"mac":"aa:bb:cc:dd:ee:ff","hostname":"tv","ipv4":"192.168.1.50","detected_at":"2026-10-12T12:00:00Z","acknowledged_at":"2026-10-12T13:00:00Z"}`))
	})

	alert, err := cl.GetV1Alert("7")
	if err != nil {
		t.Fatalf("GetV1Alert: %v", err)
	}
	if path != "/api/v1/alerts/7" {
		t.Fatalf("unexpected path %q", path)
	}
	if alert.ID != 7 || alert.Acknowledged == nil || *alert.Acknowledged != "2026-10-12T13:00:00Z" {
		t.Fatalf("unexpected alert %+v", alert)
	}
}

func TestGetV1AlertNotFound(t *testing.T) {
	cl := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no such alert"}`))
	})

	_, err := cl.GetV1Alert("999")
	if err == nil || !strings.Contains(err.Error(), "no such alert") {
		t.Fatalf("expected no-such-alert error, got %v", err)
	}
}

func TestAcknowledgeV1Alert(t *testing.T) {
	var method, path string
	var body []byte
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"id":7,"mac":"aa:bb:cc:dd:ee:ff","hostname":"tv","ipv4":"192.168.1.50","detected_at":"2026-10-12T12:00:00Z","acknowledged_at":"2026-10-12T13:00:00Z"}`))
	})

	alert, err := cl.AcknowledgeV1Alert("7")
	if err != nil {
		t.Fatalf("AcknowledgeV1Alert: %v", err)
	}
	if method != http.MethodPatch || path != "/api/v1/alerts/7" {
		t.Fatalf("expected PATCH alert endpoint, got %s %s", method, path)
	}
	if string(body) != `{"acknowledged":true}` {
		t.Fatalf("unexpected acknowledge body %q", body)
	}
	if alert.ID != 7 || alert.Acknowledged == nil {
		t.Fatalf("unexpected alert %+v", alert)
	}
}
