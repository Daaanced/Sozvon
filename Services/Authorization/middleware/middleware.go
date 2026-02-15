// Authorization/middleware/middleware.go
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"Authorization/config"
)

// Logging middleware для логирования HTTP запросов
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		log.Printf(
			"[%s] %s %s - Status: %d - Duration: %v - IP: %s",
			r.Method,
			r.RequestURI,
			r.Proto,
			lrw.statusCode,
			time.Since(start),
			r.RemoteAddr,
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Recovery middleware для обработки паник
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS middleware для настройки CORS
func CORS(cfg config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Устанавливаем CORS заголовки
			if len(cfg.AllowedOrigins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", cfg.AllowedOrigins[0])
			}

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Обработка preflight запросов
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit можно добавить позже
