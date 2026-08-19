package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const maxTrackedRateLimitClients = 10000

type IPRateLimiter struct {
	mu          sync.Mutex
	clients     map[string]rateLimitClient
	maxRequests int
	window      time.Duration
	now         func() time.Time
	lastCleanup time.Time
}

type rateLimitClient struct {
	windowStarted time.Time
	requests      int
}

func NewIPRateLimiter(maxRequests int, window time.Duration) *IPRateLimiter {
	now := time.Now()
	return &IPRateLimiter{
		clients: make(map[string]rateLimitClient), maxRequests: maxRequests,
		window: window, now: time.Now, lastCleanup: now,
	}
}

func (limiter *IPRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := limiter.allow(requestClientIP(r))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
			response.Error(w, http.StatusTooManyRequests, "请求次数过多")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (limiter *IPRateLimiter) allow(ip string) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	now := limiter.now()
	if now.Sub(limiter.lastCleanup) >= limiter.window {
		for key, client := range limiter.clients {
			if now.Sub(client.windowStarted) >= limiter.window {
				delete(limiter.clients, key)
			}
		}
		limiter.lastCleanup = now
	}
	client, exists := limiter.clients[ip]
	if !exists {
		if len(limiter.clients) >= maxTrackedRateLimitClients {
			return false, limiter.window
		}
		limiter.clients[ip] = rateLimitClient{windowStarted: now, requests: 1}
		return true, 0
	}
	if now.Sub(client.windowStarted) >= limiter.window {
		limiter.clients[ip] = rateLimitClient{windowStarted: now, requests: 1}
		return true, 0
	}
	if client.requests >= limiter.maxRequests {
		return false, limiter.window - now.Sub(client.windowStarted)
	}
	client.requests++
	limiter.clients[ip] = client
	return true, 0
}

func clientIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	return remoteAddress
}

func requestClientIP(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	return clientIP(r.RemoteAddr)
}
