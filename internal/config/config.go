package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds Blog service configuration loaded from environment variables.
type Config struct {
	AppEnv   string
	Service  string
	GRPCPort int

	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseSSLMode  string

	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	ShutdownTimeout time.Duration
}

// Load reads and validates required environment variables.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		Service: getEnv("SERVICE_NAME", "blog-service"),
	}

	var err error
	cfg.GRPCPort, err = getEnvInt("GRPC_PORT", 50052)
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseHost = os.Getenv("DATABASE_HOST")
	cfg.DatabaseName = os.Getenv("DATABASE_NAME")
	cfg.DatabaseUser = os.Getenv("DATABASE_USER")
	cfg.DatabasePassword = os.Getenv("DATABASE_PASSWORD")
	cfg.DatabaseSSLMode = getEnv("DATABASE_SSLMODE", "disable")

	cfg.DatabasePort, err = getEnvInt("DATABASE_PORT", 5432)
	if err != nil {
		return Config{}, err
	}

	cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}
	cfg.DBConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout, err = getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	required := map[string]string{
		"DATABASE_HOST":     c.DatabaseHost,
		"DATABASE_NAME":     c.DatabaseName,
		"DATABASE_USER":     c.DatabaseUser,
		"DATABASE_PASSWORD": c.DatabasePassword,
	}
	for k, v := range required {
		if v == "" {
			return fmt.Errorf("missing required env: %s", k)
		}
	}
	if c.GRPCPort <= 0 {
		return fmt.Errorf("GRPC_PORT must be positive")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}
