package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kubedoio/n-kudo/internal/controlplane/backup"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration from environment
	config := loadConfig()

	// Create backup manager
	manager := backup.NewManager(config.Backup, logger)

	// Setup cron scheduler
	schedule := getEnv("BACKUP_SCHEDULE", "0 2 * * *") // Default: daily at 2 AM
	logger.WithField("schedule", schedule).Info("Starting backup scheduler")

	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))))

	// Add backup job
	_, err := c.AddFunc(schedule, func() {
		logger.Info("Running scheduled backup")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.BackupTimeout)*time.Minute)
		defer cancel()

		backupPath, err := manager.Backup(ctx)
		if err != nil {
			logger.WithError(err).Error("Backup failed")
			return
		}
		logger.WithField("path", backupPath).Info("Backup completed successfully")

		// Run cleanup after successful backup
		if err := manager.CleanupOldBackups(ctx); err != nil {
			logger.WithError(err).Warn("Cleanup old backups failed")
		} else {
			logger.Info("Cleanup old backups completed")
		}
	})
	if err != nil {
		logger.WithError(err).Fatal("Failed to add backup job to scheduler")
	}

	// Start scheduler
	c.Start()
	logger.Info("Backup scheduler started")

	// Handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("Shutdown signal received, stopping scheduler...")

	// Stop scheduler gracefully
	stopCtx := c.Stop()
	<-stopCtx.Done()

	logger.Info("Backup scheduler stopped")
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Backup        backup.Config
	BackupTimeout int
}

func loadConfig() SchedulerConfig {
	retentionDays, _ := strconv.Atoi(getEnv("BACKUP_RETENTION_DAYS", "30"))
	compress := getEnv("BACKUP_COMPRESS", "true") == "true"
	encrypt := getEnv("BACKUP_ENCRYPT", "false") == "true"
	timeout, _ := strconv.Atoi(getEnv("BACKUP_TIMEOUT_MINUTES", "60"))

	return SchedulerConfig{
		Backup: backup.Config{
			DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nkudo?sslmode=disable"),
			BackupDir:     getEnv("BACKUP_DIR", "/var/backups/n-kudo"),
			RetentionDays: retentionDays,
			Compress:      compress,
			Encrypt:       encrypt,
			EncryptionKey: getEnv("BACKUP_ENCRYPTION_KEY", ""),
			S3Bucket:      getEnv("BACKUP_S3_BUCKET", ""),
			S3Endpoint:    getEnv("BACKUP_S3_ENDPOINT", ""),
		},
		BackupTimeout: timeout,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
