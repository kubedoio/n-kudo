package controlplane

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/kubedoio/n-kudo/internal/controlplane/grpc"
	"github.com/kubedoio/n-kudo/internal/controlplane/secrets"
)

type Config struct {
	ListenAddr           string
	DatabaseURL          string
	AdminKey             string
	DefaultTokenTTL      time.Duration
	AgentCertTTL         time.Duration
	HeartbeatInterval    time.Duration
	PlanLeaseTTL         time.Duration
	MaxPlansPerHeartbeat int
	OfflineAfter         time.Duration
	OfflineSweepInterval time.Duration
	RequirePersistentPKI bool
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	ShutdownTimeout      time.Duration
	CACommonName         string
	RateLimit            RateLimitConfig
	// Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	AppBaseURL   string // Base URL for links in emails (e.g., https://app.nkudo.io)
	// Audit configuration
	AuditVerifyInterval time.Duration // Interval for background audit chain verification
	// Secret store configuration
	SecretStore secrets.SecretStore
	// gRPC server configuration
	GRPC grpc.Config
}

func LoadConfig() Config {
	// Initialize secret store
	secretStore, err := secrets.NewStoreFromEnv()
	if err != nil {
		log.Printf("[config] Failed to create secret store, falling back to env: %v", err)
		secretStore = secrets.NewEnvSecretStore()
	}

	// Log which secret store type is being used
	if _, ok := secretStore.(*secrets.EnvSecretStore); ok {
		log.Println("[config] Using environment variable secret store")
	} else if _, ok := secretStore.(*secrets.HashiCorpVaultStore); ok {
		log.Println("[config] Using HashiCorp Vault secret store")
	} else if _, ok := secretStore.(*secrets.AWSSecretsManagerStore); ok {
		log.Println("[config] Using AWS Secrets Manager store")
	}

	cfg := Config{
		ListenAddr:           env("CONTROL_PLANE_ADDR", ":8443"),
		DefaultTokenTTL:      envDuration("DEFAULT_ENROLLMENT_TTL", 15*time.Minute),
		AgentCertTTL:         envDuration("AGENT_CERT_TTL", 24*time.Hour),
		HeartbeatInterval:    envDuration("HEARTBEAT_INTERVAL", 15*time.Second),
		PlanLeaseTTL:         envDuration("PLAN_LEASE_TTL", 45*time.Second),
		MaxPlansPerHeartbeat: envInt("MAX_PENDING_PLANS", 2),
		OfflineAfter:         envDuration("HEARTBEAT_OFFLINE_AFTER", 60*time.Second),
		OfflineSweepInterval: envDuration("OFFLINE_SWEEP_INTERVAL", 15*time.Second),
		RequirePersistentPKI: envBool("REQUIRE_PERSISTENT_PKI", false),
		ReadTimeout:          envDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:         envDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:          envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:      envDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		CACommonName:         env("CA_COMMON_NAME", "n-kudo-mvp1-agent-ca"),
		RateLimit:            DefaultRateLimitConfig(),
		// Email config - non-sensitive values from env
		SMTPHost:   env("SMTP_HOST", ""),
		SMTPPort:   envInt("SMTP_PORT", 587),
		SMTPUser:   env("SMTP_USER", ""),
		SMTPFrom:   env("SMTP_FROM", "noreply@nkudo.io"),
		AppBaseURL: env("APP_BASE_URL", "http://localhost:3000"),
		// Audit configuration (default 5 minutes)
		AuditVerifyInterval: envDuration("AUDIT_VERIFY_INTERVAL", 5*time.Minute),
		// Store reference
		SecretStore: secretStore,
		// gRPC configuration
		GRPC: grpc.Config{
			Enabled:    envBool("GRPC_ENABLED", false),
			ListenAddr: env("GRPC_LISTEN_ADDR", ":50051"),
			TLSConfig: grpc.TLSConfig{
				CertFile: env("GRPC_TLS_CERT_FILE", ""),
				KeyFile:  env("GRPC_TLS_KEY_FILE", ""),
				CAFile:   env("GRPC_TLS_CA_FILE", ""),
			},
		},
	}

	// Load sensitive values from secret store with env fallback
	cfg.DatabaseURL = getSecret(secretStore, "database/url", "DATABASE_URL",
		"postgres://nkudo:nkudo@localhost:5432/nkudo?sslmode=disable")
	cfg.AdminKey = getSecret(secretStore, "admin/key", "ADMIN_KEY", "dev-admin-key")
	cfg.SMTPPassword = getSecret(secretStore, "smtp/password", "SMTP_PASSWORD", "")

	return cfg
}

// getSecret retrieves a secret from the secret store with environment fallback
func getSecret(store secrets.SecretStore, key, envKey, defaultValue string) string {
	// Try secret store first
	if store != nil {
		if value, err := store.Get(key); err == nil && value != "" {
			return value
		}
	}

	// Fall back to environment variable
	if value := os.Getenv(envKey); value != "" {
		return value
	}

	return defaultValue
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

func envInt(k string, fallback int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(k string, fallback bool) bool {
	if v := os.Getenv(k); v != "" {
		switch v {
		case "1", "true", "TRUE", "yes", "YES", "on", "ON":
			return true
		case "0", "false", "FALSE", "no", "NO", "off", "OFF":
			return false
		}
	}
	return fallback
}
