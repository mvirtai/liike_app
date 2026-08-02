package config_test

import (
	"testing"

	"liike_app/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any environment variables that might be set
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("APP_ENV", "")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port '8080', got '%s'", cfg.Port)
	}

	if cfg.DatabaseURL != "liike.db" {
		t.Errorf("expected default DatabaseURL 'liike.db', got '%s'", cfg.DatabaseURL)
	}

	if cfg.JWTSecret != "dev-secret-key-change-in-production-123456" {
		t.Errorf("expected default JWTSecret 'dev-secret-key-change-in-production-123456', got '%s'", cfg.JWTSecret)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected default Environment 'development', got '%s'", cfg.Environment)
	}
}

func TestLoad_CustomEnvVars(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "test.db")
	t.Setenv("JWT_SECRET", "custom-secret-key")
	t.Setenv("APP_ENV", "production")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got '%s'", cfg.Port)
	}

	if cfg.DatabaseURL != "test.db" {
		t.Errorf("expected DatabaseURL 'test.db', got '%s'", cfg.DatabaseURL)
	}

	if cfg.JWTSecret != "custom-secret-key" {
		t.Errorf("expected JWTSecret 'custom-secret-key', got '%s'", cfg.JWTSecret)
	}

	if cfg.Environment != "production" {
		t.Errorf("expected Environment 'production', got '%s'", cfg.Environment)
	}
}
