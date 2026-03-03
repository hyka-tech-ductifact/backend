package persistence

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresConnection() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Validate required env vars
	for key, val := range map[string]string{
		"DB_HOST": host, "DB_PORT": port, "DB_USER": user, "DB_PASSWORD": password, "DB_NAME": dbname,
	} {
		if val == "" {
			return nil, fmt.Errorf("required environment variable %s is not set — check your .env file", key)
		}
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := db.AutoMigrate(&UserModel{}); err != nil {
			log.Printf("Failed to migrate database: %v", err)
		}
	}

	return db, nil
}
