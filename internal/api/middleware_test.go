package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoverer(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("critical unexpected condition!")
	})

	middleware := Recoverer(nil)
	wrapped := middleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 status on panic recovery, got %d", rec.Code)
	}
}

func TestRequestLogger(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	middleware := RequestLogger(nil)
	wrapped := middleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/logged", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202 status, got %d", rec.Code)
	}
}
