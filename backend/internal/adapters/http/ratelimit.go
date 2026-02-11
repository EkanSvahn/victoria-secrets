package http

import (
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"victora-secret-code/backend/internal/metrics"
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	ratePerS float64
	burst    float64
	opCount  int
}

func NewLimiter(rpm int, burst int) *Limiter {
	return &Limiter{
		buckets:  map[string]*tokenBucket{},
		ratePerS: float64(rpm) / 60.0,
		burst:    float64(burst),
	}
}

func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{tokens: l.burst, lastRefill: now}
		l.buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * l.ratePerS
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}
	bucket.lastRefill = now

	if bucket.tokens < 1.0 {
		l.cleanupExpired(now)
		return false
	}
	bucket.tokens -= 1.0
	l.cleanupExpired(now)
	return true
}

func (l *Limiter) cleanupExpired(now time.Time) {
	l.opCount++
	if l.opCount%128 != 0 {
		return
	}
	for key, bucket := range l.buckets {
		if now.Sub(bucket.lastRefill) > 30*time.Minute {
			delete(l.buckets, key)
		}
	}
}

func rateLimit(limiter *Limiter, counters *metrics.Counters, resolveClientIP func(*http.Request) string) func(http.Handler) http.Handler {
	if resolveClientIP == nil {
		resolveClientIP = remoteIP
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeKey, protected := classifyProtectedRoute(r.Method, r.URL.Path)
			if !protected {
				next.ServeHTTP(w, r)
				return
			}
			key := resolveClientIP(r) + "|" + routeKey
			if !limiter.Allow(key, time.Now()) {
				if counters != nil {
					counters.IncRateLimited()
				}
				w.Header().Set("Retry-After", "60")
				writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func classifyProtectedRoute(method, rawPath string) (string, bool) {
	p := path.Clean(strings.TrimSpace(rawPath))
	if p == "." {
		p = "/"
	}
	if method == http.MethodPost && p == "/api/v1/secrets" {
		return "POST /api/v1/secrets", true
	}
	const secretsPrefix = "/api/v1/secrets/"
	if strings.HasPrefix(p, secretsPrefix) {
		rest := strings.TrimPrefix(p, secretsPrefix)
		if rest == "" || strings.Contains(rest, "/") {
			if method == http.MethodPost && strings.HasSuffix(p, "/consume") {
				parts := strings.Split(strings.TrimPrefix(p, secretsPrefix), "/")
				if len(parts) == 2 && parts[1] == "consume" && parts[0] != "" {
					return "POST /api/v1/secrets/{id}/consume", true
				}
			}
			return "", false
		}
		if method == http.MethodGet {
			return "GET /api/v1/secrets/{id}", true
		}
	}
	return "", false
}
