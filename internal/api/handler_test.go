package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/jaskiratmiglani07/url-shortener/internal/service"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
)

func setupTestApp() (*Handler, *http.ServeMux, *store.MemoryStore) {
	memStore := store.NewMemoryStore()
	svc := service.NewShortenerService(memStore, "http://localhost:8080")
	handler := NewHandler(svc, nil)
	router := NewRouter(handler, nil)
	return handler, router, memStore
}

func TestHandler_Shorten(t *testing.T) {
	_, router, _ := setupTestApp()

	// 1. Success case
	payload := model.ShortenRequest{
		URL: "https://example.org/docs",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp model.ShortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.OriginalURL != payload.URL {
		t.Errorf("expected %s, got %s", payload.URL, resp.OriginalURL)
	}
	if resp.ShortCode == "" {
		t.Errorf("expected non-empty ShortCode")
	}

	// 2. Invalid URL
	invalidPayload := model.ShortenRequest{
		URL: "ftp://not-supported",
	}
	body, _ = json.Marshal(invalidPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request, got %d", rec.Code)
	}

	// 3. Custom Alias
	aliasPayload := model.ShortenRequest{
		URL:         "https://wikipedia.org",
		CustomAlias: "wiki-main",
	}
	body, _ = json.Marshal(aliasPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d", rec.Code)
	}

	// 4. Duplicate Alias -> 409 Conflict
	req = httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409 Conflict, got %d", rec.Code)
	}
}

func TestHandler_RedirectAndAnalytics(t *testing.T) {
	_, router, _ := setupTestApp()

	// Create link
	payload := model.ShortenRequest{
		URL:         "https://news.ycombinator.com",
		CustomAlias: "hn-news",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create URL: %d", rec.Code)
	}

	// Perform redirect
	redirectReq := httptest.NewRequest(http.MethodGet, "/hn-news", nil)
	redirectRec := httptest.NewRecorder()
	router.ServeHTTP(redirectRec, redirectReq)

	if redirectRec.Code != http.StatusFound {
		t.Errorf("expected 302 Found, got %d", redirectRec.Code)
	}
	location := redirectRec.Header().Get("Location")
	if location != "https://news.ycombinator.com" {
		t.Errorf("expected Location https://news.ycombinator.com, got %s", location)
	}

	// Preview endpoint
	previewReq := httptest.NewRequest(http.MethodGet, "/preview/hn-news", nil)
	previewRec := httptest.NewRecorder()
	router.ServeHTTP(previewRec, previewReq)

	if previewRec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", previewRec.Code)
	}

	// Analytics endpoint
	analyticsReq := httptest.NewRequest(http.MethodGet, "/analytics/hn-news", nil)
	analyticsRec := httptest.NewRecorder()
	router.ServeHTTP(analyticsRec, analyticsReq)

	if analyticsRec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", analyticsRec.Code)
	}

	var analytics model.AnalyticsResponse
	if err := json.NewDecoder(analyticsRec.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode analytics: %v", err)
	}
	if analytics.ShortCode != "hn-news" {
		t.Errorf("expected short code hn-news, got %s", analytics.ShortCode)
	}
}

func TestHandler_NotFound(t *testing.T) {
	_, router, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/unknown-code-xyz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", rec.Code)
	}
}

func TestHandler_Health(t *testing.T) {
	_, router, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
}
