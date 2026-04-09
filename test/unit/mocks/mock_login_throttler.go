package mocks

// MockLoginThrottler implements ports.LoginThrottler for testing.
type MockLoginThrottler struct {
	IsBlockedFn     func(key string) bool
	RecordFailureFn func(key string)
	ResetFn         func(key string)
	StopFn          func()
}

func (m *MockLoginThrottler) IsBlocked(key string) bool {
	if m.IsBlockedFn != nil {
		return m.IsBlockedFn(key)
	}
	return false
}

func (m *MockLoginThrottler) RecordFailure(key string) {
	if m.RecordFailureFn != nil {
		m.RecordFailureFn(key)
	}
}

func (m *MockLoginThrottler) Reset(key string) {
	if m.ResetFn != nil {
		m.ResetFn(key)
	}
}

func (m *MockLoginThrottler) Stop() {
	if m.StopFn != nil {
		m.StopFn()
	}
}
