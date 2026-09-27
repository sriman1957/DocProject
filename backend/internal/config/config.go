package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	HTTPAddr string
	JWTSecret string
	JWTAccessTokenTTL  time.Duration

	DBHost     string
	DBPort     uint16
	DBName     string
	DBUser     string
	DBPassword string
}

func Load() (Config, error) {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load(".env")
	}

	cfg := Config{
		AppEnv:     getEnv("APP_ENV", ""),
		HTTPAddr:   getEnv("HTTP_ADDR", ""),
		DBHost:     getEnv("DB_HOST", ""),
		DBName:     getEnv("DB_NAME", ""),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		JWTSecret: getEnv("JWT_SECRET", ""),
	}

	ttl, err := time.ParseDuration(
		getEnv("JWT_ACCESS_TOKEN_TTL", "15m"),
	)

	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf(
			"JWT_ACCESS_TOKEN_TTL must be a positive duration",
		)
	}
	cfg.JWTAccessTokenTTL = ttl

	port, err := strconv.ParseUint(getEnv("DB_PORT", ""), 10, 16)
	if err != nil || port == 0 {
		return Config{}, fmt.Errorf("DB_PORT must be a valid port between 1 and 65535")
	}
	cfg.DBPort = uint16(port)

	if cfg.AppEnv == "" {
		return Config{}, fmt.Errorf("APP_ENV is required")
	}
	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR is required")
	}
	if cfg.DBHost == "" || cfg.DBName == "" || cfg.DBUser == "" || cfg.DBPassword == "" {
		return Config{}, fmt.Errorf("database configuration is incomplete")
	}

	switch cfg.AppEnv {
	case "development", "production", "test":
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, production, or test")
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf(
			"JWT_SECRET must be at least 32 bytes",
		)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
