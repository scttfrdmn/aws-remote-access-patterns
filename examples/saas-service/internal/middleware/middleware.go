// Package middleware provides HTTP middleware for the SaaS service
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// requestIDKey is the context key for request IDs
type requestIDKey struct{}

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// Chain applies multiple middleware functions to a handler
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// RequestID adds a unique request ID to each request
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Add to response headers
			w.Header().Set("X-Request-ID", requestID)

			// Add to context
			ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Logging provides structured logging for HTTP requests
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log request start
			logger.Debug("Request started",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()))

			defer func() {
				duration := time.Since(start)

				// Log request completion
				logger.Info("Request completed",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", wrapped.statusCode),
					slog.Duration("duration", duration),
					slog.Int64("duration_ms", duration.Milliseconds()))
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}

// Recovery recovers from panics and returns a 500 error
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r.Context())

					logger.Error("Request panic recovered",
						slog.String("request_id", requestID),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.Any("panic", err),
						slog.String("stack", string(debug.Stack())))

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds Cross-Origin Resource Sharing headers
func CORS() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Allow specific origins or all origins in development
			// In production, you should restrict this to your actual domains
			if origin != "" && (strings.Contains(origin, "localhost") || 
				strings.Contains(origin, "127.0.0.1") ||
				strings.HasSuffix(origin, ".yourdomain.com")) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Security adds security headers
func Security() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit provides basic rate limiting (in-memory, not suitable for production)
func RateLimit(requestsPerMinute int) Middleware {
	// This is a simple in-memory rate limiter
	// For production, use Redis or a proper rate limiting service
	clients := make(map[string][]time.Time)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			now := time.Now()
			
			// Clean old entries
			if requests, exists := clients[clientIP]; exists {
				var validRequests []time.Time
				for _, reqTime := range requests {
					if now.Sub(reqTime) < time.Minute {
						validRequests = append(validRequests, reqTime)
					}
				}
				clients[clientIP] = validRequests
			}
			
			// Check rate limit
			if len(clients[clientIP]) >= requestsPerMinute {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			// Add current request
			clients[clientIP] = append(clients[clientIP], now)
			
			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		return requestID
	}
	return ""
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req-%s", hex.EncodeToString(bytes))
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if there are multiple
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check for X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}