package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoLeech/GoLeech/internal/task"
)

func TestWebServerEndpoints(t *testing.T) {
	task.InitManager()
	server := StartServer(0)
	defer server.Shutdown(nil)

	// 1. Health check
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	server.handleHealthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /healthz, got %d", w.Code)
	}

	// 2. Stats API
	reqStats := httptest.NewRequest("GET", "/api/stats", nil)
	wStats := httptest.NewRecorder()
	server.handleAPIStats(wStats, reqStats)

	if wStats.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /api/stats, got %d", wStats.Code)
	}
	var stats map[string]interface{}
	if err := json.Unmarshal(wStats.Body.Bytes(), &stats); err != nil {
		t.Errorf("failed to parse stats json: %v", err)
	}
	if _, ok := stats["goroutines"]; !ok {
		t.Errorf("expected goroutines in stats response")
	}

	// 3. Dashboard HTML
	reqDash := httptest.NewRequest("GET", "/", nil)
	wDash := httptest.NewRecorder()
	server.handleDashboard(wDash, reqDash)

	if wDash.Code != http.StatusOK {
		t.Errorf("expected 200 OK for dashboard, got %d", wDash.Code)
	}
	bodyStr := wDash.Body.String()
	if len(bodyStr) == 0 || bodyStr[:15] != "<!DOCTYPE html>" {
		t.Errorf("expected HTML doctype in dashboard response")
	}
}
