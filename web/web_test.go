package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebEmbedding(t *testing.T) {
	// 1. IndexHandler
	h := IndexHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "GoLink Shortener") {
		t.Errorf("expected body to contain 'GoLink Shortener'")
	}

	// 2. StaticHandler
	staticH, err := StaticHandler()
	if err != nil {
		t.Fatalf("failed to create static handler: %v", err)
	}

	staticReq := httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil)
	staticRec := httptest.NewRecorder()

	staticH.ServeHTTP(staticRec, staticReq)

	if staticRec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for static CSS, got %d", staticRec.Code)
	}
}
