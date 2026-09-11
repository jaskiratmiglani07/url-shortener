package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLimiter_MemoryAllow(t *testing.T) {
	ctx := context.Background()
	// Limiter with rps=2, burst=2
	limiter := NewLimiter(nil, 2, 2, true, nil)

	ip := "192.168.1.50"

	// First request: allowed
	allowed, _ := limiter.Allow(ctx, ip)
	if !allowed {
		t.Errorf("expected 1st request to be allowed")
	}

	// Second request: allowed (burst=2)
	allowed, _ = limiter.Allow(ctx, ip)
	if !allowed {
		t.Errorf("expected 2nd request to be allowed")
	}

	// Third request: rejected (exceeded burst)
	allowed, _ = limiter.Allow(ctx, ip)
	if allowed {
		t.Errorf("expected 3rd request to be rejected")
	}

	// Different IP should still be allowed
	allowed, _ = limiter.Allow(ctx, "10.0.0.1")
	if !allowed {
		t.Errorf("expected different IP to be allowed")
	}
}

func TestLimiter_Disabled(t *testing.T) {
	ctx := context.Background()
	limiter := NewLimiter(nil, 1, 1, false, nil)

	for i := 0; i < 10; i++ {
		allowed, _ := limiter.Allow(ctx, "127.0.0.1")
		if !allowed {
			t.Errorf("disabled limiter should allow all requests, rejected at %d", i)
		}
	}
}

func TestLimiter_Middleware(t *testing.T) {
	limiter := NewLimiter(nil, 2, 2, true, nil)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrapped := limiter.Middleware(dummyHandler)

	// 1st request -> 200
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "1.2.3.4:12345"
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-RateLimit-Limit") != "2" {
		t.Errorf("expected header X-RateLimit-Limit 2, got %s", rec1.Header().Get("X-RateLimit-Limit"))
	}

	// 2nd request -> 200
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "1.2.3.4:12345"
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}

	// 3rd request -> 429 Too Many Requests
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.RemoteAddr = "1.2.3.4:12345"
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", rec3.Code)
	}
	if rec3.Header().Get("Retry-After") != "1" {
		t.Errorf("expected Retry-After 1, got %s", rec3.Header().Get("Retry-After"))
	}

	// Health endpoint should bypass rate limit even when exhausted
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	reqHealth.RemoteAddr = "1.2.3.4:12345"
	recHealth := httptest.NewRecorder()
	wrapped.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected health check to bypass rate limit with 200, got %d", recHealth.Code)
	}
}

func TestExtractIP(t *testing.T) {
	// X-Forwarded-For with multiple IPs
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	if ip := ExtractIP(r1); ip != "203.0.113.195" {
		t.Errorf("expected 203.0.113.195, got %s", ip)
	}

	// X-Real-IP
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set("X-Real-IP", "198.51.100.22")
	if ip := ExtractIP(r2); ip != "198.51.100.22" {
		t.Errorf("expected 198.51.100.22, got %s", ip)
	}

	// RemoteAddr fallback with port
	r3 := httptest.NewRequest(http.MethodGet, "/", nil)
	r3.RemoteAddr = "192.0.2.1:54321"
	if ip := ExtractIP(r3); ip != "192.0.2.1" {
		t.Errorf("expected 192.0.2.1, got %s", ip)
	}
}
