package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr      string
	RedisURL        string
	MaxBodyBytes    int64
	MaxTTLSeconds   int64
	IDLengthBytes   int
	RequestTimeout  time.Duration
	AllowedOrigin   string
	TrustedProxyCID string
	RateLimitRPM    int
	RateLimitBurst  int
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:      getenv("LISTEN_ADDR", ":8080"),
		RedisURL:        getenv("REDIS_URL", "redis://redis:6379/0"),
		AllowedOrigin:   getenv("ALLOWED_ORIGIN", "http://localhost:5173"),
		TrustedProxyCID: getenv("TRUSTED_PROXY_CIDR", ""),
	}

	var err error
	cfg.MaxBodyBytes, err = getenvInt64("MAX_BODY_BYTES", 4*1024*1024)
	if err != nil {
		return Config{}, fmt.Errorf("MAX_BODY_BYTES: %w", err)
	}
	cfg.MaxTTLSeconds, err = getenvInt64("MAX_TTL_SECONDS", 86400)
	if err != nil {
		return Config{}, fmt.Errorf("MAX_TTL_SECONDS: %w", err)
	}
	cfg.IDLengthBytes, err = getenvInt("ID_LENGTH_BYTES", 24)
	if err != nil {
		return Config{}, fmt.Errorf("ID_LENGTH_BYTES: %w", err)
	}
	if cfg.IDLengthBytes < 16 || cfg.IDLengthBytes > 64 {
		return Config{}, fmt.Errorf("ID_LENGTH_BYTES must be between 16 and 64")
	}
	cfg.RateLimitRPM, err = getenvInt("RATE_LIMIT_RPM", 60)
	if err != nil {
		return Config{}, fmt.Errorf("RATE_LIMIT_RPM: %w", err)
	}
	if cfg.RateLimitRPM < 1 || cfg.RateLimitRPM > 10000 {
		return Config{}, fmt.Errorf("RATE_LIMIT_RPM must be between 1 and 10000")
	}
	cfg.RateLimitBurst, err = getenvInt("RATE_LIMIT_BURST", 20)
	if err != nil {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST: %w", err)
	}
	if cfg.RateLimitBurst < 1 || cfg.RateLimitBurst > 10000 {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be between 1 and 10000")
	}
	timeoutMs, err := getenvInt64("REQUEST_TIMEOUT_MS", 5000)
	if err != nil {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT_MS: %w", err)
	}
	cfg.RequestTimeout = time.Duration(timeoutMs) * time.Millisecond

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func getenvInt64(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
