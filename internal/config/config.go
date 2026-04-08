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
	App       App
	Database  Database
	JWT       JWT
	Log       Log
	CORS      CORS
	RateLimit RateLimit
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

// Load reads all configuration from environment variables.
// Required variables panic if missing — call this at startup
// before any other initialization.
func Load() Config {
	return Config{
		App: App{
			Host: optional("APP_HOST", "localhost"),
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
			TokenDuration:        parseDuration(optional("JWT_TOKEN_DURATION", "15m")),
			RefreshTokenDuration: parseDuration(optional("JWT_REFRESH_TOKEN_DURATION", "168h")),
		},
		Log: Log{
			Level:  optional("LOG_LEVEL", "info"),
			Format: optional("LOG_FORMAT", "text"),
		},
		CORS: CORS{
			AllowedOrigins: parseList(required("CORS_ORIGINS")),
		},
		RateLimit: RateLimit{
			IPMaxRequests:   parseInt(optional("RATE_LIMIT_IP_MAX", "100")),
			IPWindow:        parseDuration(optional("RATE_LIMIT_IP_WINDOW", "1m")),
			UserMaxRequests: parseInt(optional("RATE_LIMIT_USER_MAX", "60")),
			UserWindow:      parseDuration(optional("RATE_LIMIT_USER_WINDOW", "1m")),
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

// optional reads an environment variable, returning defaultValue if empty or unset.
func optional(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
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
