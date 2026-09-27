package db

import (
	"testing"
	"time"

	"docproject/backend/internal/config"
)

func TestNewPoolConfig(t *testing.T) {
	cfg := config.Config{
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "docproject",
		DBUser:     "docproject_app",
		DBPassword: "test-password",
	}

	poolConfig, err := newPoolConfig(cfg)
	if err != nil {
		t.Fatalf("newPoolConfig() error = %v", err)
	}

	if poolConfig.ConnConfig.Host != "localhost" {
		t.Errorf(
			"Host = %q, want %q",
			poolConfig.ConnConfig.Host,
			"localhost",
		)
	}

	if poolConfig.ConnConfig.Port != 5432 {
		t.Errorf(
			"Port = %d, want %d",
			poolConfig.ConnConfig.Port,
			5432,
		)
	}

	if poolConfig.ConnConfig.Database != "docproject" {
		t.Errorf(
			"Database = %q, want %q",
			poolConfig.ConnConfig.Database,
			"docproject",
		)
	}

	if poolConfig.ConnConfig.User != "docproject_app" {
		t.Errorf(
			"User = %q, want %q",
			poolConfig.ConnConfig.User,
			"docproject_app",
		)
	}

	if poolConfig.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want 10", poolConfig.MaxConns)
	}

	if poolConfig.MinConns != 0 {
		t.Errorf("MinConns = %d, want 0", poolConfig.MinConns)
	}

	if poolConfig.MaxConnLifetime != time.Hour {
		t.Errorf(
			"MaxConnLifetime = %v, want %v",
			poolConfig.MaxConnLifetime,
			time.Hour,
		)
	}

	if poolConfig.MaxConnIdleTime != 30*time.Minute {
		t.Errorf(
			"MaxConnIdleTime = %v, want %v",
			poolConfig.MaxConnIdleTime,
			30*time.Minute,
		)
	}
}
