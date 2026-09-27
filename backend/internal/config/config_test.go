package config

import (
	"os"
	"testing"
	"time"
)

func setValidEnv(t *testing.T) {
	t.Helper()

	env := map[string]string{
		"APP_ENV":              "test",
		"HTTP_ADDR":            ":8080",
		"DB_HOST":              "localhost",
		"DB_PORT":              "5432",
		"DB_NAME":              "docproject",
		"DB_USER":              "docproject_app",
		"DB_PASSWORD":          "test-password",
		"JWT_SECRET":           "test-only-secret-at-least-32-bytes-long",
		"JWT_ACCESS_TOKEN_TTL": "15m",
	}

	for key, value := range env {
		t.Setenv(key, value)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.AppEnv != "test" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "test")
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}

	if cfg.DBPort != 5432 {
		t.Errorf("DBPort = %d, want 5432", cfg.DBPort)
	}

	if cfg.DBName != "docproject" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "docproject")
	}

	if cfg.JWTAccessTokenTTL != 15*time.Minute {
		t.Errorf("JWTAccessTokenTTL = %v, want 15m", cfg.JWTAccessTokenTTL)
	}
}

func TestLoad_MissingRequiredSetting(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_HOST", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when DB_HOST is missing")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_PORT", "not-a-port")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail for an invalid DB_PORT")
	}
}

func TestLoad_UnsupportedEnvironment(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ENV", "staging")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should reject an unsupported APP_ENV")
	}
}

func TestLoad_DoesNotExposePasswordInErrors(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_PASSWORD", "test-secret")
	t.Setenv("DB_PORT", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return an error")
	}

	if err.Error() == "test-secret" {
		t.Fatal("error should not expose the database password")
	}
}

func TestLoad_EnvironmentOverridesDotEnv(t *testing.T) {
	setValidEnv(t)

	t.Setenv("APP_ENV", "test")
	t.Setenv("DB_NAME", "environment_database")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.DBName != "environment_database" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "environment_database")
	}
}

func TestGetEnv_Fallback(t *testing.T) {
	const key = "DOCPROJECT_TEST_MISSING_SETTING"

	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv() failed: %v", err)
	}

	got := getEnv(key, "fallback")
	if got != "fallback" {
		t.Errorf("getEnv() = %q, want %q", got, "fallback")
	}
}

func TestLoad_RejectsMissingJWTSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should reject a missing JWT_SECRET")
	}
}

func TestLoad_RejectsInvalidJWTAccessTokenTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_ACCESS_TOKEN_TTL", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should reject an invalid JWT_ACCESS_TOKEN_TTL")
	}
}
