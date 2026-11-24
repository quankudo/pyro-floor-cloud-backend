package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	AppPort            string
	CORSAllowedOrigins []string
	Auth               AuthConfig
	DB                 DatabaseConfig
}

type AuthConfig struct {
	AccessTokenSecret  []byte
	RefreshTokenSecret []byte
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		Auth: AuthConfig{
			AccessTokenSecret:  []byte(os.Getenv("AUTH_ACCESS_TOKEN_SECRET")),
			RefreshTokenSecret: []byte(os.Getenv("AUTH_REFRESH_TOKEN_SECRET")),
		},
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "app"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "appdb"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
	defaultOrigins := ""
	if cfg.AppEnv == "development" {
		defaultOrigins = "http://localhost:5173"
	}
	cfg.CORSAllowedOrigins = parseCSV(getEnv("CORS_ALLOWED_ORIGINS", defaultOrigins))
	if len(cfg.CORSAllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one trusted origin")
	}
	if len(cfg.Auth.AccessTokenSecret) < 32 ||
		len(cfg.Auth.RefreshTokenSecret) < 32 {
		return Config{}, fmt.Errorf("AUTH_ACCESS_TOKEN_SECRET and AUTH_REFRESH_TOKEN_SECRET must each be at least 32 bytes")
	}
	if string(cfg.Auth.AccessTokenSecret) == string(cfg.Auth.RefreshTokenSecret) {
		return Config{}, fmt.Errorf("access and refresh token secrets must be different")
	}

	var err error
	cfg.Auth.AccessTokenTTL, err = getDuration(
		"AUTH_ACCESS_TOKEN_TTL",
		60*time.Minute,
	)
	if err != nil {
		return Config{}, err
	}
	cfg.Auth.RefreshTokenTTL, err = getDuration(
		"AUTH_REFRESH_TOKEN_TTL",
		24*time.Hour,
	)
	if err != nil {
		return Config{}, err
	}
	if cfg.Auth.AccessTokenTTL <= 0 ||
		cfg.Auth.RefreshTokenTTL <= cfg.Auth.AccessTokenTTL {
		return Config{}, fmt.Errorf(
			"AUTH_REFRESH_TOKEN_TTL must be greater than AUTH_ACCESS_TOKEN_TTL and both must be positive",
		)
	}

	cfg.DB.MaxConns, err = getInt32("DB_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	cfg.DB.MinConns, err = getInt32("DB_MIN_CONNS", 0)
	if err != nil {
		return Config{}, err
	}

	if cfg.AppPort == "" {
		return Config{}, fmt.Errorf("APP_PORT must not be empty")
	}

	if cfg.DB.Host == "" || cfg.DB.User == "" || cfg.DB.Name == "" {
		return Config{}, fmt.Errorf(
			"DB_HOST, DB_USER and DB_NAME must not be empty",
		)
	}

	if cfg.DB.Password == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD must not be empty")
	}

	if cfg.DB.MaxConns < 1 ||
		cfg.DB.MinConns < 0 ||
		cfg.DB.MinConns > cfg.DB.MaxConns {
		return Config{}, fmt.Errorf("invalid database connection pool settings")
	}

	if err := validateDBPort(cfg.DB.Port); err != nil {
		return Config{}, err
	}

	switch cfg.DB.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return Config{}, fmt.Errorf("invalid DB_SSLMODE: %q", cfg.DB.SSLMode)
	}

	return cfg, nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return duration, nil
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

func getInt32(key string, fallback int32) (int32, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return int32(n), nil
}

func validateDBPort(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid DB_PORT: %q", value)
	}
	return nil
}
