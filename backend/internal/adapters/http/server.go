package http

import (
	"log/slog"
	"net/http"
	"time"

	"victora-secret-code/backend/internal/app"
	"victora-secret-code/backend/internal/config"
	"victora-secret-code/backend/internal/metrics"
)

func NewServer(cfg config.Config, service *app.Service, readiness ReadinessCheck) *http.Server {
	mux := http.NewServeMux()
	counters := metrics.NewCounters()
	handler := NewHandler(service, RequestLimits{
		MaxMetaBytes:     cfg.MaxMetaBytes,
		MaxCipherBytes:   cfg.MaxCipherBytes,
		MaxFileBytes:     cfg.MaxFileBytes,
		AllowedFileMIMEs: cfg.AllowedFileMIMEs,
		MaxTTLSeconds:    cfg.MaxTTLSeconds,
		MaxViews:         cfg.MaxViews,
		RequirePassword:  cfg.RequirePassword,
	}, counters)
	handler.SetReadinessCheck(readiness)
	handler.RegisterRoutes(mux, cfg.MetricsEnabled)
	limiter := NewLimiter(cfg.RateLimitRPM, cfg.RateLimitBurst)
	resolveClientIP := newClientIPResolver(cfg.TrustedProxyCID)

	stack := chain(
		mux,
		withRequestID,
		requestLogger(slog.Default(), resolveClientIP),
		securityHeaders,
		cors(cfg.AllowedOrigins),
		rateLimit(limiter, counters, resolveClientIP),
		withBodyLimit(cfg.MaxBodyBytes),
		withTimeout(cfg.RequestTimeout),
	)

	readTimeout := cfg.RequestTimeout + (5 * time.Second)
	if readTimeout < 15*time.Second {
		readTimeout = 15 * time.Second
	}
	writeTimeout := cfg.RequestTimeout + (10 * time.Second)
	if writeTimeout < 30*time.Second {
		writeTimeout = 30 * time.Second
	}

	return &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           stack,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       30 * time.Second,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
}
