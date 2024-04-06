package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterMiddleware(t *testing.T) {
	// Create a RateLimiter instance
	config := RateLimitConfig{
		Endpoints: make(map[string]map[string]*TokenBucketConfig),
	}
	rl := NewRateLimiter(config, time.Minute)
	rl.StartCleanup()
	defer rl.StopCleanup()

	// Create a test HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	// Wrap the test handler with the rate limiter middleware
	limitedHandler := rl.Middleware(handler)

	// Create a test server with the limitedHandler
	ts := httptest.NewServer(limitedHandler)
	defer ts.Close()

	// Send multiple requests and check the responses
	for i := 0; i < 20; i++ {
		resp, err := http.Get(ts.URL)
		if err != nil {
			t.Fatalf("Error making GET request: %v", err)
		}

		// Check response status code
		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limit exceeded, test passed
			break
		}

		// Read the response body
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Error reading response body: %v", err)
		}

		// Check response body
		expectedBody := "Hello, World!"
		if string(body) != expectedBody {
			t.Fatalf("Unexpected response body: got %s, want %s", string(body), expectedBody)
		}

		// Close response body
		err = resp.Body.Close()
		if err != nil {
			t.Fatalf("Error closing response body: %v", err)
		}
	}

}
