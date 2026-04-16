package persistence

import (
	"fmt"
	"log/slog"
	"strings"

	"ductifact/internal/config"
	"ductifact/internal/infrastructure/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresConnection(cfg config.Database, logLevel string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(parseGormLogLevel(logLevel)),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run versioned SQL migrations (replaces GORM AutoMigrate).
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying *sql.DB: %w", err)
	}

	if err := migrations.Run(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations applied successfully")

	// Seed predefined piece definitions (upsert — safe to run every startup).
	if err := SeedPredefinedPieceDefinitions(db); err != nil {
		return nil, fmt.Errorf("failed to seed predefined piece definitions: %w", err)
	}

	return db, nil
}

// parseGormLogLevel maps the application log level string to GORM's logger level.
func parseGormLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return logger.Info // GORM has no debug; Info is the most verbose
	case "info":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	case "silent":
		return logger.Silent
	default:
		return logger.Info
	}
}
