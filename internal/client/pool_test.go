package client

import (
	"net/http"
	"testing"
)

func TestGetV1Pool(t *testing.T) {
	var path string
	cl := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"subnet":"192.168.1.0/24","total":254,"used":57,"free":197,"utilization":22.4}`))
	})

	pool, err := cl.GetV1Pool()
	if err != nil {
		t.Fatalf("GetV1Pool: %v", err)
	}
	if path != "/api/v1/pool" {
		t.Fatalf("unexpected path %q", path)
	}
	if pool.Subnet != "192.168.1.0/24" || pool.Total != 254 || pool.Used != 57 || pool.Free != 197 || pool.Utilization != 22.4 {
		t.Fatalf("unexpected pool %+v", pool)
	}
}
