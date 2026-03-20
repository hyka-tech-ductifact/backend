package persistence

import (
	"fmt"
	"log/slog"
	"strings"

	"ductifact/internal/config"

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

	if err := db.AutoMigrate(&UserModel{}, &ClientModel{}); err != nil {
		slog.Warn("auto-migration failed", "error", err)
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
