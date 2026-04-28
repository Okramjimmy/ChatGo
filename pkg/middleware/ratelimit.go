package middleware

import (
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	apperr "github.com/okrammeitei/chatgo/pkg/errors"
)

// RateLimiter returns a global (per-server) token-bucket rate limiter middleware.
// For per-IP limiting, wrap with a sync.Map of limiters keyed by IP.
func RateLimiter(rps float64, burst int, log *zap.Logger) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				log.Warn("rate limit exceeded", zap.String("path", r.URL.Path))
				w.Header().Set("Retry-After", "1")
				WriteError(w, &apperr.AppError{Code: http.StatusTooManyRequests, Message: "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
