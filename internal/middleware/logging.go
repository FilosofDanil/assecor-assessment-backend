package middleware

import (
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Logging gibt eine Middleware zurück, die jede Anfrage mit Methode, Pfad, Statuscode, Dauer und Request-ID
func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("anfrage",
				zap.String("request_id", chimw.GetReqID(r.Context())),
				zap.String("methode", r.Method),
				zap.String("pfad", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Duration("dauer", time.Since(start)),
			)
		})
	}
}
