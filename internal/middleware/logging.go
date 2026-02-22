package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// statusWriter kapselt http. ResponseWriter, um den gesetzten HTTP-Statuscode nach dem Schreiben der Antwort abzufragen.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// Logging gibt eine Middleware zurück, die jede Anfrage mit Methode, Pfad, Statuscode und Dauer strukturiert protokolliert.
func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}

			next.ServeHTTP(sw, r)

			logger.Info("anfrage",
				zap.String("methode", r.Method),
				zap.String("pfad", r.URL.Path),
				zap.Int("status", sw.code),
				zap.Duration("dauer", time.Since(start)),
			)
		})
	}
}
