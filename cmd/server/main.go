package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/api"
	"github.com/jaskiratmiglani07/url-shortener/internal/config"
	"github.com/jaskiratmiglani07/url-shortener/internal/database"
	"github.com/jaskiratmiglani07/url-shortener/internal/ratelimit"
	"github.com/jaskiratmiglani07/url-shortener/internal/service"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
	"github.com/jaskiratmiglani07/url-shortener/web"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting URL shortener service...")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 2. Initialize primary database store (PostgreSQL with in-memory fallback)
	var primaryStore store.URLStore
	pgStore, err := store.NewPostgresStore(cfg)
	if err != nil {
		logger.Warn("could not connect to postgresql; falling back to in-memory store", "error", err)
		primaryStore = store.NewMemoryStore()
	} else {
		logger.Info("connected to postgresql database successfully")
		primaryStore = pgStore
	}
	defer primaryStore.Close()

	// 3. Initialize Redis client (with graceful fallback on failure)
	redisClient, err := database.NewRedisClient(cfg, logger)
	if err != nil {
		logger.Warn("redis cache is offline; operating with direct database lookups", "error", err)
	} else {
		defer redisClient.Close()
	}

	// 4. Wrap store with Redis cache-aside layer
	cachedStore := store.NewCachedStore(primaryStore, redisClient, cfg.CacheTTL, logger)
	defer cachedStore.Close()

	// 5. Initialize service layer
	shortenerSvc := service.NewShortenerService(cachedStore, cfg.BaseURL)

	// 6. Initialize rate limiter
	limiter := ratelimit.NewLimiter(redisClient, cfg.RateLimitRPS, cfg.RateLimitBurst, cfg.RateLimitEnabled, logger)

	// 7. Initialize HTTP handlers and router
	staticHandler, err := web.StaticHandler()
	if err != nil {
		logger.Warn("failed to initialize embedded static handler", "error", err)
	}

	handler := api.NewHandler(shortenerSvc, logger)
	router := api.NewRouter(handler, staticHandler)

	// 8. Chain middlewares: Panic Recovery -> Request Logging -> Rate Limiter -> Router
	var finalHandler http.Handler = router
	finalHandler = limiter.Middleware(finalHandler)
	finalHandler = api.RequestLogger(logger)(finalHandler)
	finalHandler = api.Recoverer(logger)(finalHandler)

	// 9. Configure HTTP Server
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      finalHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 10. Start server in background
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", serverAddr, "base_url", cfg.BaseURL)
		serverErrors <- srv.ListenAndServe()
	}()

	// 11. Graceful shutdown listening for OS interrupt signals
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server encountered fatal error", "error", err)
		}
	case sig := <-shutdownSignal:
		logger.Info("shutdown signal received; commencing graceful shutdown", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server forced to shutdown", "error", err)
			_ = srv.Close()
		}
		logger.Info("server stopped gracefully")
	}
}
