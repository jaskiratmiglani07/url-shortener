package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/api"
	"github.com/redis/go-redis/v9"
)

// Limiter controls request throughput based on client IP.
type Limiter struct {
	rdb     *redis.Client
	rps     int
	burst   int
	enabled bool
	logger  *slog.Logger

	// In-memory fallback
	mu      sync.Mutex
	buckets map[string]*clientBucket
}

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
}

// NewLimiter creates a rate limiter with Redis backing and in-memory fallback.
func NewLimiter(rdb *redis.Client, rps, burst int, enabled bool, logger *slog.Logger) *Limiter {
	if logger == nil {
		logger = slog.Default()
	}
	if rps <= 0 {
		rps = 10
	}
	if burst < rps {
		burst = rps * 2
	}

	return &Limiter{
		rdb:     rdb,
		rps:     rps,
		burst:   burst,
		enabled: enabled,
		logger:  logger,
		buckets: make(map[string]*clientBucket),
	}
}

// Allow checks if the client IP is permitted to make a request.
// Returns allowed (bool) and remaining quota.
func (l *Limiter) Allow(ctx context.Context, ip string) (bool, int) {
	if !l.enabled {
		return true, l.burst
	}

	// Try Redis-backed rate limiting first
	if l.rdb != nil {
		allowed, remaining, err := l.allowRedis(ctx, ip)
		if err == nil {
			return allowed, remaining
		}
		// Redis error -> Fall back to in-memory limiter
		l.logger.Warn("redis rate limiting error; falling back to in-memory limiter", "error", err)
	}

	// In-memory token bucket fallback
	return l.allowMemory(ip)
}

func (l *Limiter) allowRedis(ctx context.Context, ip string) (bool, int, error) {
	now := time.Now().Unix()
	key := fmt.Sprintf("ratelimit:%s:%d", ip, now)

	// Pipeline INCR and EXPIRE for atomic 1-second fixed-window rate limiting
	pipe := l.rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := int(incrCmd.Val())
	if count > l.rps {
		return false, 0, nil
	}

	return true, l.rps - count, nil
}

func (l *Limiter) allowMemory(ip string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.buckets[ip]
	if !exists {
		bucket = &clientBucket{
			tokens:     float64(l.burst),
			lastRefill: now,
		}
		l.buckets[ip] = bucket
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * float64(l.rps)
	if bucket.tokens > float64(l.burst) {
		bucket.tokens = float64(l.burst)
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true, int(bucket.tokens)
	}

	return false, 0
}

// Middleware creates an HTTP middleware that enforces rate limiting per client IP.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for health check and static endpoints
		path := r.URL.Path
		if path == "/health" || path == "/healthz" || strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		ip := ExtractIP(r)
		allowed, remaining := l.Allow(r.Context(), ip)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.rps))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			w.Header().Set("Retry-After", "1")
			api.RespondError(w, http.StatusTooManyRequests, "rate limit exceeded: please slow down")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ExtractIP retrieves the client IP address from proxy headers or remote address.
func ExtractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if ip != "" {
			return ip
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}
