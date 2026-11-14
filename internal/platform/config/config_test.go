package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("READ_TIMEOUT", "")

	cfg := Load()
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.LogLevel)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Fatalf("expected default read timeout 5s, got %v", cfg.ReadTimeout)
	}
}

func TestLoadOverridesAndInvalidDuration(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("READ_TIMEOUT", "15s")
	t.Setenv("WRITE_TIMEOUT", "invalid")

	cfg := Load()
	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %q", cfg.Port)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Fatalf("expected read timeout 15s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Fatalf("expected fallback write timeout 10s on invalid input, got %v", cfg.WriteTimeout)
	}
}
