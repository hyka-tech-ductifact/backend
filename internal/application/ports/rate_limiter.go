package ports

// RateLimiter is the outbound port for controlling request rates.
// It tracks how many requests a given key (IP address, user ID, etc.)
// has made within a time window and decides whether to allow more.
//
// This is defined as an interface so the HTTP layer doesn't depend on
// a specific implementation — it can be backed by in-memory counters,
// Redis, or any other store.
type RateLimiter interface {
	// Allow checks whether the given key is allowed to make a request.
	// Returns true if the request is within the rate limit, false if it should be rejected.
	Allow(key string) bool

	// Stop terminates any background goroutines (cleanup, etc.).
	// Call this during graceful shutdown.
	Stop()
}
