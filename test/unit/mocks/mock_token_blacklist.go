package mocks

import "time"

// MockTokenBlacklist implements ports.TokenBlacklist for testing.
type MockTokenBlacklist struct {
	AddFn           func(token string, expiry time.Duration)
	IsBlacklistedFn func(token string) bool
}

func (m *MockTokenBlacklist) Add(token string, expiry time.Duration) {
	if m.AddFn != nil {
		m.AddFn(token, expiry)
	}
}

func (m *MockTokenBlacklist) IsBlacklisted(token string) bool {
	if m.IsBlacklistedFn != nil {
		return m.IsBlacklistedFn(token)
	}
	return false
}
