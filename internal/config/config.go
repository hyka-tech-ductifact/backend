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
	Redis         Redis
	JWT           JWT
	Log           Log
	CORS          CORS
	RateLimit     RateLimit
	LoginThrottle LoginThrottle
	MinIO         MinIO
	SMTP          SMTP
	OneTimeTokens OneTimeTokens
}

// OneTimeTokens holds TTL configuration for one-time-use tokens.
type OneTimeTokens struct {
	EmailVerificationTTL time.Duration // How long email verification tokens remain valid (e.g. "24h")
	PasswordResetTTL     time.Duration // How long password reset tokens remain valid (e.g. "1h")
	VerificationBaseURL  string        // Base URL for verification links (e.g. "https://app.ductifact.com")
}

// MinIO holds S3-compatible object storage settings.
type MinIO struct {
	Endpoint  string // e.g. "localhost:9000"
	AccessKey string
	SecretKey string
	Bucket    string // e.g. "ductifact"
	UseSSL    bool
}

// SMTP holds email delivery settings.
// In development, point to MailPit (localhost:1025) which captures all emails.
// In production, point to a real provider (SendGrid, SES, etc.).
type SMTP struct {
	Host     string // SMTP server hostname (e.g. "smtp.sendgrid.net", "localhost" for MailPit)
	Port     int    // SMTP server port (e.g. 587 for TLS, 1025 for MailPit)
	UseAuth  bool   // Whether to authenticate with the SMTP server (false for MailPit)
	Username string // AUTH username ("apikey" for SendGrid)
	Password string // AUTH password / API key
	From     string // sender address (e.g. "noreply@ductifact.com")
}

// Redis holds Redis connection settings.
// Redis is always the primary backend for distributed state.
// If unavailable at runtime, the system degrades to in-memory adapters.
type Redis struct {
	Host     string // hostname (e.g. "localhost")
	Port     string // port (e.g. "6379")
	UseAuth  bool   // whether to authenticate with the Redis server
	Password string // AUTH password (only used when UseAuth is true)
	DB       int    // database number (0-15)
}

// Addr returns the Redis address in host:port format.
func (r Redis) Addr() string {
	return r.Host + ":" + r.Port
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
		Redis: Redis{
			Host:     required("REDIS_HOST"),
			Port:     required("REDIS_PORT"),
			UseAuth:  parseBool(required("REDIS_AUTH")),
			Password: required("REDIS_PASSWORD"),
			DB:       parseInt(required("REDIS_DB")),
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
			Endpoint:  required("MINIO_HOST") + ":" + required("MINIO_API_PORT"),
			AccessKey: required("MINIO_ROOT_USER"),
			SecretKey: required("MINIO_ROOT_PASSWORD"),
			Bucket:    required("MINIO_BUCKET"),
			UseSSL:    parseBool(required("MINIO_USE_SSL")),
		},
		SMTP: SMTP{
			Host:     required("SMTP_HOST"),
			Port:     parseInt(required("SMTP_PORT")),
			UseAuth:  parseBool(required("SMTP_AUTH")),
			Username: required("SMTP_USERNAME"),
			Password: required("SMTP_PASSWORD"),
			From:     required("SMTP_FROM"),
		},
		OneTimeTokens: OneTimeTokens{
			EmailVerificationTTL: parseDuration(required("VERIFICATION_EMAIL_TOKEN_TTL")),
			PasswordResetTTL:     parseDuration(required("VERIFICATION_PASSWORD_RESET_TOKEN_TTL")),
			VerificationBaseURL:  required("VERIFICATION_BASE_URL"),
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
