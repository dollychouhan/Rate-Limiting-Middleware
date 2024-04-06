package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limit settings for different endpoints and client IP addresses.
type RateLimitConfig struct {
	Endpoints map[string]map[string]*TokenBucketConfig // Map of endpoint to client IP to TokenBucketConfig struct
}

// TokenBucketConfig holds configuration for the token bucket algorithm.
type TokenBucketConfig struct {
	Capacity int
	Rate     rate.Limit
	Bucket   *TokenBucket // Pointer to the actual TokenBucket instance
}

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	Capacity int
	Tokens   int
	LastFill time.Time
	mutex    *sync.Mutex
}

// NewTokenBucket creates a new TokenBucket instance with given capacity and rate.
func NewTokenBucket(capacity int, rate rate.Limit) *TokenBucket {
	mutex := sync.Mutex{}
	return &TokenBucket{
		Capacity: capacity,
		Tokens:   capacity, // Initially filled to capacity
		mutex:    &mutex,
	}
}

// Allow checks if the request can be allowed based on the token bucket algorithm.
func (tb *TokenBucket) Allow() bool {
	now := time.Now()

	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	elapsed := now.Sub(tb.LastFill)
	newTokens := int(float64(elapsed) / float64(time.Second) * float64(tb.Capacity))
	tb.Tokens = min(tb.Capacity, tb.Tokens+newTokens) // Refill tokens

	if tb.Tokens > 0 {
		tb.Tokens--
		tb.LastFill = now
		return true
	}
	return false
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimiter holds rate limit records and configurations.
type RateLimiter struct {
	config          RateLimitConfig
	mutex           sync.Mutex
	cleanupInterval time.Duration // Interval for cleanup process
	cleanupStop     chan struct{} // Channel to stop the cleanup process
}

// NewRateLimiter creates a new RateLimiter with given configuration and cleanup interval.
func NewRateLimiter(config RateLimitConfig, cleanupInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		config:          config,
		mutex:           sync.Mutex{},
		cleanupInterval: cleanupInterval,
		cleanupStop:     make(chan struct{}),
	}
}

// Middleware intercepts incoming requests and enforces rate limits using token bucket.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := r.URL.Path
		clientIP := r.RemoteAddr // Assuming client IP is present in the RemoteAddr field

		rl.mutex.Lock()
		defer rl.mutex.Unlock()

		// Initialize rate record if not exists
		if _, ok := rl.config.Endpoints[endpoint]; !ok {
			rl.config.Endpoints[endpoint] = make(map[string]*TokenBucketConfig)
		}
		if _, ok := rl.config.Endpoints[endpoint][clientIP]; !ok {
			tb := NewTokenBucket(10, rate.Limit(100*time.Millisecond)) // Set capacity and rate based on your configuration
			tbConfig := &TokenBucketConfig{
				Capacity: 10,                      // Set default capacity
				Rate:     rate.Limit(time.Second), // Set default rate
				Bucket:   tb,
			}
			rl.config.Endpoints[endpoint][clientIP] = tbConfig
		}

		// Check if request allowed by token bucket
		if !rl.config.Endpoints[endpoint][clientIP].Bucket.Allow() {
			log.Printf("Request from %s to %s blocked due to rate limiting", clientIP, endpoint)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// StartCleanup starts a background goroutine to periodically clean up expired entries.
func (rl *RateLimiter) StartCleanup() {
	go func() {
		ticker := time.NewTicker(rl.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-rl.cleanupStop:
				return
			case <-ticker.C:
				if err := rl.cleanupExpiredEntries(); err != nil {
					log.Printf("Error cleaning up expired entries: %v", err)
				}
			}
		}
	}()
}

// StopCleanup stops the background goroutine for cleaning up entries.
func (rl *RateLimiter) StopCleanup() {
	close(rl.cleanupStop)
}

// cleanupExpiredEntries removes entries from the config that have expired.
func (rl *RateLimiter) cleanupExpiredEntries() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	for endpoint, endpointMap := range rl.config.Endpoints {
		for clientIP, tbConfig := range endpointMap {
			// Check if token bucket has expired
			elapsed := now.Sub(tbConfig.Bucket.LastFill)
			if elapsed > rl.cleanupInterval {
				delete(endpointMap, clientIP) // Remove expired entry
				if len(endpointMap) == 0 {
					delete(rl.config.Endpoints, endpoint) // Remove empty endpoint map
				}
			}
		}
	}
	return nil
}

func main() {
	// Example usage
	config := RateLimitConfig{
		Endpoints: make(map[string]map[string]*TokenBucketConfig),
	}
	rl := NewRateLimiter(config, time.Minute) // Set cleanup interval as per your requirement
	rl.StartCleanup()
	defer rl.StopCleanup()

	http.Handle("/", rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello, World!")) // Write response
		if err != nil {
			log.Printf("Error writing response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
