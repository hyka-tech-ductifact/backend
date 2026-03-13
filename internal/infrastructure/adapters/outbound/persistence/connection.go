package persistence

import (
	"fmt"
	"log/slog"

	"ductifact/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresConnection(cfg config.Database) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if cfg.AutoMigrate {
		if err := db.AutoMigrate(&UserModel{}, &ClientModel{}); err != nil {
			slog.Warn("auto-migration failed", "error", err)
		}
	}

	return db, nil
}
