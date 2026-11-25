package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Define client
type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Define rate limiter
type RateLimiter struct {
	clients map[string]*client
	mu      sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*client),
	}
	go rl.cleanupClients()
	return rl
}

// Remove clients that sent no request for 3 minutes (Start this every minute)
func (rl *RateLimiter) cleanupClients() {
	for {
		time.Sleep(1 * time.Minute)
		rl.mu.Lock()
		for ip, c := range rl.clients {
			if time.Since(c.lastSeen) > 3*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Check if client is saved
func (rl *RateLimiter) GetClient(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, exists := rl.clients[ip]
	if !exists {
		// Allow 1 requests per second, with a burst of 2
		limiter := rate.NewLimiter(1, 2)
		rl.clients[ip] = &client{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get IP
		ip := getIP(r)

		// Get limit
		limiter := rl.GetClient(ip)

		// Check
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests - Please slow down.", http.StatusTooManyRequests)
			return
		}

		// Next
		next.ServeHTTP(w, r)
	})
}

// Get clients IP
func getIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
