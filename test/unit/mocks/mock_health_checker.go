package mocks

import "context"

// MockHealthChecker implements ports.HealthChecker for testing.
// Configure PingFn per test to control the behavior.
type MockHealthChecker struct {
	PingFn func(ctx context.Context) error
}

func (m *MockHealthChecker) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return nil
}
