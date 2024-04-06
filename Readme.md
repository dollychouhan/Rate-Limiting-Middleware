Components:

Middleware Function: The Middleware function of the RateLimiter type acts as the intercepting point for incoming requests. It extracts the endpoint and client IP from the request, retrieves the corresponding TokenBucketConfig, and checks if the request is allowed by the token bucket algorithm.

Data Structures and Algorithms:

RateLimitConfig: This struct holds a nested map to store rate limit configurations for different endpoints (string) and client IP addresses (string). Each entry maps to a TokenBucketConfig struct.
TokenBucketConfig: This struct defines the configuration for the token bucket algorithm, including Capacity (maximum tokens allowed), Rate (rate of token refill), and a pointer to the actual TokenBucket instance (Bucket).
TokenBucket: This struct implements the token bucket algorithm. It has fields for Capacity, the current number of tokens (Tokens), and the LastFill time (used to calculate token refill). The Allow method checks if a request can be granted based on available tokens and refills the bucket if necessary.
Configuration Options:

The RateLimitConfig structure allows specifying default or custom rate limits for different endpoints and client IPs.
The NewTokenBucket function within the Middleware function sets default values for Capacity and Rate if no specific configuration exists for a client IP within an endpoint.
Concurrency Safety:

The RateLimiter uses a sync.Mutex (mutex) to ensure synchronized access to the config map when storing or retrieving rate limit entries. This prevents race conditions during concurrent requests.
Expiry Mechanism:

The cleanupExpiredEntries function periodically removes entries from the config map that haven't been used (haven't had their LastFill time updated) within the configured cleanupInterval. This ensures outdated rate limit records are cleaned up, freeing resources.
Explanation:

The code defines structs (RateLimitConfig, TokenBucketConfig, TokenBucket) to manage rate limit configurations and the token bucket algorithm.
The NewRateLimiter function creates a new RateLimiter instance with the provided configuration and cleanup interval.
The Middleware function intercepts requests, retrieves the corresponding TokenBucketConfig, and calls the Allow method of the TokenBucket to check if the request can be processed based on available tokens.
If allowed, the request proceeds to the next handler in the chain.
If blocked due to rate limiting, an error message is logged, and a response indicating "Rate limit exceeded" is sent back to the client.
The StartCleanup and StopCleanup functions manage a background goroutine that periodically calls cleanupExpiredEntries to remove unused entries from the configuration map.
The cleanupExpiredEntries function iterates through the config map and removes entries where the elapsed time since the LastFill is greater than the cleanupInterval.
