package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipRecord struct {
	count       int
	windowStart time.Time
}

// RateLimiter implements a sliding-window rate limiter per IP address.
type RateLimiter struct {
	mu      sync.Mutex
	records map[string]*ipRecord
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter that allows 'limit' requests per 'window' per IP.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		records: make(map[string]*ipRecord),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, record := range rl.records {
			if now.Sub(record.windowStart) > rl.window {
				delete(rl.records, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks whether a request from the given IP is allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	record, exists := rl.records[ip]

	if !exists || now.Sub(record.windowStart) > rl.window {
		rl.records[ip] = &ipRecord{count: 1, windowStart: now}
		return true
	}

	record.count++
	return record.count <= rl.limit
}

// clientIP extracts the client IP from the request, respecting proxy headers.
func clientIP(r *http.Request) string {
	// X-Forwarded-For (may contain multiple IPs: client, proxy1, proxy2)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip := strings.Split(forwarded, ",")[0]
		return strings.TrimSpace(ip)
	}
	// X-Real-IP (set by nginx)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	// Fallback: RemoteAddr (strip port)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Middleware wraps an http.Handler with IP-based rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.Allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests, please try again later", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
