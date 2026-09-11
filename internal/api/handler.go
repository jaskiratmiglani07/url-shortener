package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/jaskiratmiglani07/url-shortener/internal/service"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
)

// Handler provides HTTP handlers for the URL shortener service.
type Handler struct {
	shortener *service.ShortenerService
	logger    *slog.Logger
}

// NewHandler creates a new Handler instance.
func NewHandler(shortener *service.ShortenerService, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		shortener: shortener,
		logger:    logger,
	}
}

// Shorten handles URL creation requests.
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Limit request body to 1MB to prevent large payload attacks
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	var req model.ShortenRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			RespondError(w, http.StatusBadRequest, "request body cannot be empty")
			return
		}
		RespondError(w, http.StatusBadRequest, "invalid json payload: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.shortener.Shorten(ctx, &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyURL),
			errors.Is(err, service.ErrInvalidURL),
			errors.Is(err, service.ErrInvalidScheme),
			errors.Is(err, service.ErrMissingHost),
			errors.Is(err, service.ErrURLTooLong),
			errors.Is(err, service.ErrInvalidAlias),
			errors.Is(err, service.ErrReservedAlias),
			errors.Is(err, service.ErrExpiredTimestamp):
			RespondError(w, http.StatusBadRequest, err.Error())
			return
		case errors.Is(err, store.ErrAlreadyExists):
			RespondError(w, http.StatusConflict, "custom alias is already taken")
			return
		default:
			h.logger.Error("failed to shorten url", "error", err)
			RespondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	RespondJSON(w, http.StatusCreated, resp)
}

// Redirect handles short code redirection.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := extractShortCode(r)
	if code == "" {
		RespondError(w, http.StatusBadRequest, "short code is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	u, err := h.shortener.Resolve(ctx, code)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			RespondError(w, http.StatusNotFound, "short url not found")
			return
		case errors.Is(err, store.ErrURLExpired):
			RespondError(w, http.StatusGone, "short url has expired")
			return
		default:
			h.logger.Error("failed to resolve short url", "code", code, "error", err)
			RespondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	// Record click asynchronously so user redirection is never blocked
	go func(urlID int64) {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer bgCancel()
		if err := h.shortener.RecordClick(bgCtx, urlID); err != nil {
			h.logger.Warn("failed to record click event", "url_id", urlID, "error", err)
		}
	}(u.ID)

	// Instruct browser/intermediaries not to cache redirect so click analytics capture every visit
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.Redirect(w, r, u.OriginalURL, http.StatusFound)
}

// Preview retrieves URL metadata without triggering a redirect or recording a click.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := extractShortCode(r)
	if code == "" {
		RespondError(w, http.StatusBadRequest, "short code is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	u, err := h.shortener.Resolve(ctx, code)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			RespondError(w, http.StatusNotFound, "short url not found")
			return
		case errors.Is(err, store.ErrURLExpired):
			RespondError(w, http.StatusGone, "short url has expired")
			return
		default:
			h.logger.Error("failed to preview short url", "code", code, "error", err)
			RespondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	RespondJSON(w, http.StatusOK, u)
}

// Analytics retrieves link click analytics.
func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	code := extractShortCode(r)
	if code == "" {
		RespondError(w, http.StatusBadRequest, "short code is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	analytics, err := h.shortener.GetAnalytics(ctx, code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			RespondError(w, http.StatusNotFound, "short url not found")
			return
		}
		h.logger.Error("failed to fetch analytics", "code", code, "error", err)
		RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	RespondJSON(w, http.StatusOK, analytics)
}

// Health provides a basic liveness/readiness probe.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// extractShortCode extracts the short code from path parameters or URL path.
func extractShortCode(r *http.Request) string {
	if val := r.PathValue("shortCode"); val != "" {
		return val
	}
	// Fallback for manual path splitting if PathValue is empty
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
