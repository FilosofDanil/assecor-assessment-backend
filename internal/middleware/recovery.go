package middleware

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recovery gibt eine Middleware zurück, die Panics abfängt, den Stack-Trace protokolliert und dem Client einen HTTP-500-Fehler zurückgibt.
func Recovery(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic abgefangen",
						zap.Any("fehler", rec),
						zap.ByteString("stack", debug.Stack()),
					)
					http.Error(w, `{"error":"interner serverfehler"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
