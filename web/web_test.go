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

// TestAnalyticsResultVisibilityMechanism guards against the regression where
// #analytics-result is shown/hidden via classList toggling instead of
// style.display toggling. Because #analytics-result carries an inline
// style="display: none;", a CSS class-based .active rule is overridden by
// the inline style (inline styles have higher specificity), so the panel would
// never actually appear. The correct fix is to use element.style.display
// directly in JavaScript.
func TestAnalyticsResultVisibilityMechanism(t *testing.T) {
	// --- Check HTML: #analytics-result must use inline display:none ---
	h := IndexHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	htmlBody := rec.Body.String()

	// The analytics-result div must be hidden via an inline style so that
	// JS can reveal it by setting element.style.display = 'block'.
	if !strings.Contains(htmlBody, `id="analytics-result"`) {
		t.Fatal("expected #analytics-result element to be present in HTML")
	}
	// Verify the element has an inline display:none (not just a CSS class).
	if !strings.Contains(htmlBody, `id="analytics-result" style="display: none;"`) {
		t.Error(`#analytics-result must have inline style="display: none;" so that JS style.display toggling works correctly`)
	}

	// --- Check JS: must use style.display, not classList, for analytics-result ---
	staticH, err := StaticHandler()
	if err != nil {
		t.Fatalf("failed to create static handler: %v", err)
	}
	jsReq := httptest.NewRequest(http.MethodGet, "/static/js/app.js", nil)
	jsRec := httptest.NewRecorder()
	staticH.ServeHTTP(jsRec, jsReq)

	if jsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for app.js, got %d", jsRec.Code)
	}
	jsBody := jsRec.Body.String()

	// Must use direct style.display toggling for analyticsResult.
	if !strings.Contains(jsBody, "analyticsResult.style.display = 'block'") {
		t.Error("app.js must show #analytics-result via analyticsResult.style.display = 'block', not classList")
	}
	if !strings.Contains(jsBody, "analyticsResult.style.display = 'none'") {
		t.Error("app.js must hide #analytics-result via analyticsResult.style.display = 'none', not classList")
	}

	// Must NOT use classList to toggle visibility on analyticsResult,
	// as that would be overridden by the element's inline display:none.
	if strings.Contains(jsBody, "analyticsResult.classList.add('active')") {
		t.Error("app.js must not use analyticsResult.classList.add('active') — inline style overrides it")
	}
	if strings.Contains(jsBody, "analyticsResult.classList.remove('active')") {
		t.Error("app.js must not use analyticsResult.classList.remove('active') — inline style overrides it")
	}
}

