package persistence

import (
	"context"

	"gorm.io/gorm"
)

// PostgresHealthChecker implements ports.HealthChecker using GORM.
type PostgresHealthChecker struct {
	db *gorm.DB
}

// NewPostgresHealthChecker creates a new PostgresHealthChecker.
func NewPostgresHealthChecker(db *gorm.DB) *PostgresHealthChecker {
	return &PostgresHealthChecker{db: db}
}

// Ping verifies the database connection is alive.
func (h *PostgresHealthChecker) Ping(ctx context.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
