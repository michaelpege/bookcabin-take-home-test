package config

import (
	"os"
	"time"
)

// Config holds runtime configuration for the HTTP API server and dependencies.
type Config struct {
	Port             string
	LogLevel         string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ShutdownTimeout  time.Duration
	RedisAddr        string
	CacheBaseTTL     time.Duration
	CacheProviderTTL time.Duration
}

// Load reads configuration from environment variables, falling back to safe defaults.
func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		ReadTimeout:      getDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:     getDuration("WRITE_TIMEOUT", 10*time.Second),
		ShutdownTimeout:  getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		CacheBaseTTL:     getDuration("CACHE_BASE_TTL", 2*time.Minute),
		CacheProviderTTL: getDuration("CACHE_PROVIDER_TTL", 5*time.Minute),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
