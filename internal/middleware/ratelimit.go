package middleware

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimit gibt eine Middleware zurück, die eingehende Anfragen auf
func RateLimit(requestsPerSecond float64, logger *zap.Logger) func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), int(requestsPerSecond))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				logger.Warn("rate-limit überschritten",
					zap.String("remote", r.RemoteAddr),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "zu viele anfragen",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
