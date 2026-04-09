package mocks

// MockRateLimiter implements ports.RateLimiter for testing.
type MockRateLimiter struct {
	AllowFn func(key string) bool
	StopFn  func()
}

func (m *MockRateLimiter) Allow(key string) bool {
	if m.AllowFn != nil {
		return m.AllowFn(key)
	}
	return true
}

func (m *MockRateLimiter) Stop() {
	if m.StopFn != nil {
		m.StopFn()
	}
}
