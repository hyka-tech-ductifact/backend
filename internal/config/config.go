// Package config centralizes all application configuration.
//
// All configuration values are loaded once at startup via Load().
// Required values panic if not set — this is intentional to catch
// misconfigurations at deploy time, not at runtime.
//
// Currently backed by environment variables, but the interface
// is designed so the source can be swapped (e.g. encrypted files,
// Vault, SSM) without changing any consumer.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration, grouped by concern.
type Config struct {
	App           App
	Database      Database
	JWT           JWT
	Log           Log
	CORS          CORS
	RateLimit     RateLimit
	LoginThrottle LoginThrottle
	MinIO         MinIO
}

// MinIO holds S3-compatible object storage settings.
type MinIO struct {
	Endpoint  string // e.g. "localhost:9000"
	AccessKey string
	SecretKey string
	Bucket    string // e.g. "ductifact"
	UseSSL    bool
}

// App holds general application settings.
type App struct {
	Host string // Hostname/IP the API is reachable on (used by tests)
	Port string // TCP port the HTTP server listens on
}

// Database holds PostgreSQL connection settings.
type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN returns the PostgreSQL connection string.
func (d Database) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		d.Host, d.User, d.Password, d.Name, d.Port,
	)
}

// JWT holds authentication token settings.
type JWT struct {
	Secret               string        // HMAC signing key
	TokenDuration        time.Duration // How long access tokens remain valid
	RefreshTokenDuration time.Duration // How long refresh tokens remain valid
}

// Log holds logging configuration.
type Log struct {
	Level  string // debug, info, warn, error
	Format string // text or json
}

// CORS holds cross-origin resource sharing settings.
type CORS struct {
	AllowedOrigins []string
}

// RateLimit holds rate limiting configuration.
type RateLimit struct {
	IPMaxRequests   int           // Max requests per IP per window
	IPWindow        time.Duration // Time window for IP rate limiting
	UserMaxRequests int           // Max requests per authenticated user per window
	UserWindow      time.Duration // Time window for user rate limiting
}

// LoginThrottle holds brute-force protection configuration.
type LoginThrottle struct {
	MaxAttempts     int           // Max failed login attempts before lockout
	Window          time.Duration // Time window for counting failures
	LockoutDuration time.Duration // How long to lock the account after max failures
}

// Load reads all configuration from environment variables.
// Required variables panic if missing — call this at startup
// before any other initialization.
func Load() Config {
	return Config{
		App: App{
			Host: required("APP_HOST"),
			Port: required("APP_PORT"),
		},
		Database: Database{
			Host:     required("DB_HOST"),
			Port:     required("DB_PORT"),
			User:     required("DB_USER"),
			Password: required("DB_PASSWORD"),
			Name:     required("DB_NAME"),
		},
		JWT: JWT{
			Secret:               required("JWT_SECRET"),
			TokenDuration:        parseDuration(required("JWT_TOKEN_DURATION")),
			RefreshTokenDuration: parseDuration(required("JWT_REFRESH_TOKEN_DURATION")),
		},
		Log: Log{
			Level:  required("LOG_LEVEL"),
			Format: required("LOG_FORMAT"),
		},
		CORS: CORS{
			AllowedOrigins: parseList(required("CORS_ORIGINS")),
		},
		RateLimit: RateLimit{
			IPMaxRequests:   parseInt(required("RATE_LIMIT_IP_MAX")),
			IPWindow:        parseDuration(required("RATE_LIMIT_IP_WINDOW")),
			UserMaxRequests: parseInt(required("RATE_LIMIT_USER_MAX")),
			UserWindow:      parseDuration(required("RATE_LIMIT_USER_WINDOW")),
		},
		LoginThrottle: LoginThrottle{
			MaxAttempts:     parseInt(required("LOGIN_THROTTLE_MAX_ATTEMPTS")),
			Window:          parseDuration(required("LOGIN_THROTTLE_WINDOW")),
			LockoutDuration: parseDuration(required("LOGIN_THROTTLE_LOCKOUT")),
		},
		MinIO: MinIO{
			Endpoint:  required("MINIO_ENDPOINT"),
			AccessKey: required("MINIO_ACCESS_KEY"),
			SecretKey: required("MINIO_SECRET_KEY"),
			Bucket:    required("MINIO_BUCKET"),
			UseSSL:    parseBool(required("MINIO_USE_SSL")),
		},
	}
}

// --- helpers (private — swap these to change the config source) ---

// required reads an environment variable and panics if it's empty or unset.
func required(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

// parseList splits a comma-separated string into a trimmed slice.
func parseList(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseDuration parses a duration string (e.g. "24h", "30m").
// Panics if the format is invalid — this is a configuration error.
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("invalid duration %q: %v", s, err))
	}
	return d
}

// parseInt parses a string as an integer.
// Panics if the format is invalid — this is a configuration error.
func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("invalid integer %q: %v", s, err))
	}
	return n
}

// parseBool parses a string as a boolean ("true"/"false").
// Panics if the format is invalid — this is a configuration error.
func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		panic(fmt.Sprintf("invalid boolean %q: %v", s, err))
	}
	return b
}
