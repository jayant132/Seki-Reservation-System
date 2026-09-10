package config

import (
	"os"
)

// Config holds all runtime configuration, loaded from environment variables.
// Keeping this centralized (rather than reading os.Getenv scattered through
// the codebase) means every dependency's startup requirements are visible
// in one place, which matters once this runs in more than one environment.
type Config struct {
	Port          string
	DatabaseURL   string
	RedisAddr     string
	JWTSecret     string
	Env           string
	MigrationsDir string
	OTLPEndpoint  string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://seki:seki@localhost:5432/seki?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-change-me-in-production"),
		Env:           getEnv("ENV", "development"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "migrations"),
		OTLPEndpoint:  getEnv("OTLP_ENDPOINT", "localhost:4317"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
