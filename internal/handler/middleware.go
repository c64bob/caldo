package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"caldo/internal/logging"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDFromContext returns the request ID and whether it was present.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

// RequestIDMiddleware assigns a request_id to every HTTP request.
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID, err := logging.NewCorrelationID()
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SafeLoggingMiddleware logs request metadata without leaking sensitive user data.
func SafeLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
			var panicValue any

			defer func() {
				if recovered := recover(); recovered != nil {
					panicValue = recovered
					if !wrapped.wroteHeader {
						wrapped.status = http.StatusInternalServerError
					}
				}

				requestID, ok := RequestIDFromContext(r.Context())
				if !ok {
					logger.Error("http_request_missing_request_id", "error", fmt.Errorf("missing request_id"))
				} else {
					logger.Info("http_request",
						"request_id", requestID,
						"method", r.Method,
						"path", r.URL.Path,
						"status", wrapped.status,
						"duration_ms", time.Since(start).Milliseconds(),
					)
				}

				if panicValue != nil {
					panic(panicValue)
				}
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}

// RecoveryMiddleware recovers panics and returns a generic 500 response.
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := &commitTrackingResponseWriter{ResponseWriter: w}

			defer func() {
				if recover() == nil {
					return
				}

				requestID, _ := RequestIDFromContext(r.Context())
				logger.Error("http_panic_recovered",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
				)

				if wrapped.wroteHeader {
					return
				}

				wrapped.Header().Set("Content-Type", "text/plain; charset=utf-8")
				http.Error(wrapped, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}

// SecurityHeadersMiddleware sets security response headers for all requests.
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	const csp = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none';"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", csp)

			next.ServeHTTP(w, r)
		})
	}
}

// ReverseProxyAuthMiddleware requires a non-empty user value in the configured proxy auth header.
func ReverseProxyAuthMiddleware(proxyUserHeader string) func(http.Handler) http.Handler {
	normalizedHeader := strings.TrimSpace(proxyUserHeader)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			if strings.TrimSpace(r.Header.Get(normalizedHeader)) == "" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.wroteHeader = true
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

type commitTrackingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *commitTrackingResponseWriter) WriteHeader(statusCode int) {
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *commitTrackingResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
