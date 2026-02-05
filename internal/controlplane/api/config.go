package controlplane

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr      string
	DatabaseURL     string
	AdminKey        string
	DefaultTokenTTL time.Duration
	AgentCertTTL    time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	CACommonName    string
}

func LoadConfig() Config {
	return Config{
		ListenAddr:      env("CONTROL_PLANE_ADDR", ":8443"),
		DatabaseURL:     env("DATABASE_URL", "postgres://nkudo:nkudo@localhost:5432/nkudo?sslmode=disable"),
		AdminKey:        env("ADMIN_KEY", "dev-admin-key"),
		DefaultTokenTTL: envDuration("DEFAULT_ENROLLMENT_TTL", 15*time.Minute),
		AgentCertTTL:    envDuration("AGENT_CERT_TTL", 24*time.Hour),
		ReadTimeout:     envDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    envDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: envDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		CACommonName:    env("CA_COMMON_NAME", "n-kudo-mvp1-agent-ca"),
	}
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func envDuration(k string, fallback time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}
