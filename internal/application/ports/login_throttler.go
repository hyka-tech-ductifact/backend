package ports

// LoginThrottler is the outbound port for tracking failed login attempts
// and blocking accounts that have exceeded the allowed number of failures.
//
// Unlike rate limiting (which counts ALL requests), login throttling only
// counts FAILED login attempts and temporarily locks the account after
// too many failures. This is the standard defense against brute-force
// password guessing attacks.
//
// The key is typically the email address, so an attacker cannot bypass
// the protection by switching IP addresses.
type LoginThrottler interface {
	// IsBlocked returns true if the given key (email) has exceeded
	// the maximum number of failed login attempts and is temporarily locked.
	IsBlocked(key string) bool

	// RecordFailure increments the failed attempt counter for the given key.
	// After reaching the configured threshold, IsBlocked will return true
	// for the duration of the lockout window.
	RecordFailure(key string)

	// Reset clears the failure counter for the given key.
	// Call this after a successful login so the user isn't penalized
	// for past failures.
	Reset(key string)

	// Stop terminates any background goroutines (cleanup, etc.).
	Stop()
}
