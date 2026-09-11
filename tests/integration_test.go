package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/api"
	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/jaskiratmiglani07/url-shortener/internal/ratelimit"
	"github.com/jaskiratmiglani07/url-shortener/internal/service"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
	"github.com/jaskiratmiglani07/url-shortener/web"
)

func setupTestServer(rps, burst int, rateLimitEnabled bool) (http.Handler, *store.MemoryStore) {
	memStore := store.NewMemoryStore()
	cachedStore := store.NewCachedStore(memStore, nil, 10*time.Minute, nil)
	shortenerSvc := service.NewShortenerService(cachedStore, "http://localhost:8080")

	staticHandler, _ := web.StaticHandler()
	indexHandler := web.IndexHandler()
	handler := api.NewHandler(shortenerSvc, nil)
	router := api.NewRouter(handler, indexHandler, staticHandler)

	limiter := ratelimit.NewLimiter(nil, rps, burst, rateLimitEnabled, nil)

	var app http.Handler = router
	app = limiter.Middleware(app)
	app = api.RequestLogger(nil)(app)
	app = api.Recoverer(nil)(app)

	return app, memStore
}

func TestFullE2EFlow(t *testing.T) {
	app, _ := setupTestServer(50, 100, true)

	// 1. Shorten a URL
	createPayload := model.ShortenRequest{
		URL: "https://www.deepmind.com/research",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var shortenResp model.ShortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&shortenResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	shortCode := shortenResp.ShortCode
	if shortCode == "" {
		t.Fatalf("expected non-empty shortCode")
	}

	// 2. Perform 3 redirects and verify 302 and headers
	for i := 0; i < 3; i++ {
		redReq := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
		redRec := httptest.NewRecorder()
		app.ServeHTTP(redRec, redReq)

		if redRec.Code != http.StatusFound {
			t.Fatalf("redirect %d: expected 302 Found, got %d", i+1, redRec.Code)
		}
		if redRec.Header().Get("Location") != "https://www.deepmind.com/research" {
			t.Errorf("expected redirect location https://www.deepmind.com/research, got %s", redRec.Header().Get("Location"))
		}
		if redRec.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
			t.Errorf("expected no-cache header on redirect")
		}
	}

	// Wait briefly for asynchronous goroutines to record clicks
	time.Sleep(50 * time.Millisecond)

	// 3. Inspect Click Analytics via GET /api/v1/analytics/{code}
	analyticsReq := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/"+shortCode, nil)
	analyticsRec := httptest.NewRecorder()
	app.ServeHTTP(analyticsRec, analyticsReq)

	if analyticsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for analytics, got %d", analyticsRec.Code)
	}

	var analytics model.AnalyticsResponse
	if err := json.NewDecoder(analyticsRec.Body).Decode(&analytics); err != nil {
		t.Fatalf("failed to decode analytics: %v", err)
	}

	if analytics.TotalClicks != 3 {
		t.Errorf("expected 3 total clicks, got %d", analytics.TotalClicks)
	}
	if analytics.ClicksToday != 3 {
		t.Errorf("expected 3 clicks today, got %d", analytics.ClicksToday)
	}
	if analytics.ClicksLast7Days != 3 {
		t.Errorf("expected 3 clicks in last 7 days, got %d", analytics.ClicksLast7Days)
	}
	if analytics.LastClickedAt == nil {
		t.Errorf("expected LastClickedAt timestamp to be populated")
	}
	if analytics.IsExpired {
		t.Errorf("expected IsExpired to be false")
	}

	// 4. Preview endpoint GET /api/v1/preview/{code}
	prevReq := httptest.NewRequest(http.MethodGet, "/api/v1/preview/"+shortCode, nil)
	prevRec := httptest.NewRecorder()
	app.ServeHTTP(prevRec, prevReq)

	if prevRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for preview, got %d", prevRec.Code)
	}
}

func TestCustomAliasAndExpirationFlow(t *testing.T) {
	app, memStore := setupTestServer(50, 100, true)

	// 1. Create with custom alias
	mins := 60
	payload := model.ShortenRequest{
		URL:              "https://news.ycombinator.com",
		CustomAlias:      "tech-news",
		ExpiresInMinutes: &mins,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", rec.Code)
	}

	// 2. Duplicate alias should return 409 Conflict
	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewReader(body))
	dupReq.Header.Set("Content-Type", "application/json")
	dupRec := httptest.NewRecorder()
	app.ServeHTTP(dupRec, dupReq)

	if dupRec.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate alias, got %d", dupRec.Code)
	}

	// 3. Active link redirects normally
	redReq := httptest.NewRequest(http.MethodGet, "/tech-news", nil)
	redRec := httptest.NewRecorder()
	app.ServeHTTP(redRec, redReq)
	if redRec.Code != http.StatusFound {
		t.Fatalf("expected 302 Found, got %d", redRec.Code)
	}

	// 4. Manually expire the link
	past := time.Now().UTC().Add(-5 * time.Minute)
	_ = memStore.SetExpiration("tech-news", &past)

	// 5. Expired link should return 410 Gone
	expReq := httptest.NewRequest(http.MethodGet, "/tech-news", nil)
	expRec := httptest.NewRecorder()
	app.ServeHTTP(expRec, expReq)

	if expRec.Code != http.StatusGone {
		t.Errorf("expected 410 Gone for expired link, got %d", expRec.Code)
	}
}

func TestRateLimitingFlow(t *testing.T) {
	// Limiter with rps=2, burst=2
	app, _ := setupTestServer(2, 2, true)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("health check request %d should succeed", i+1)
		}
	}

	// Non-health endpoint: 1st and 2nd pass, 3rd blocked
	req1 := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	req1.RemoteAddr = "192.168.10.10:9999"
	rec1 := httptest.NewRecorder()
	app.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	req2.RemoteAddr = "192.168.10.10:9999"
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec2.Code)
	}

	// 3rd request from same IP should receive 429 Too Many Requests
	req3 := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	req3.RemoteAddr = "192.168.10.10:9999"
	rec3 := httptest.NewRecorder()
	app.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", rec3.Code)
	}
	if rec3.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After header: %s", rec3.Header().Get("Retry-After"))
	}
}
