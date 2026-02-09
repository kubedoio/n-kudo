package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds backup configuration
type Config struct {
	DatabaseURL   string
	BackupDir     string
	RetentionDays int
	Compress      bool
	Encrypt       bool
	EncryptionKey string
	S3Bucket      string
	S3Endpoint    string
}

// Manager handles database backup operations
type Manager struct {
	config Config
	logger *logrus.Logger
}

// NewManager creates a new backup manager
func NewManager(config Config, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}
	return &Manager{
		config: config,
		logger: logger,
	}
}

// Backup performs a database backup
func (m *Manager) Backup(ctx context.Context) (string, error) {
	timestamp := time.Now().UTC().Format("20060102_150405")
	backupName := fmt.Sprintf("backup_%s.sql", timestamp)
	backupPath := filepath.Join(m.config.BackupDir, backupName)

	// Ensure backup directory exists
	if err := os.MkdirAll(m.config.BackupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	// Create pg_dump command
	cmd := exec.CommandContext(ctx, "pg_dump", "-d", m.config.DatabaseURL, "-F", "p", "-f", backupPath)
	cmd.Env = os.Environ()

	m.logger.WithField("backup_path", backupPath).Info("Starting database backup")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
	}

	m.logger.WithField("backup_path", backupPath).Info("Database dump completed")

	finalPath := backupPath

	// Compress if enabled
	if m.config.Compress {
		compressedPath, err := m.compressFile(ctx, backupPath)
		if err != nil {
			_ = os.Remove(backupPath)
			return "", fmt.Errorf("compressing backup: %w", err)
		}
		_ = os.Remove(backupPath) // Remove uncompressed
		finalPath = compressedPath
		m.logger.WithField("compressed_path", finalPath).Info("Backup compressed")
	}

	// Encrypt if enabled
	if m.config.Encrypt && m.config.EncryptionKey != "" {
		encryptedPath, err := m.encryptFile(ctx, finalPath)
		if err != nil {
			_ = os.Remove(finalPath)
			return "", fmt.Errorf("encrypting backup: %w", err)
		}
		_ = os.Remove(finalPath) // Remove unencrypted
		finalPath = encryptedPath
		m.logger.WithField("encrypted_path", finalPath).Info("Backup encrypted")
	}

	// Upload to S3 if configured
	if m.config.S3Bucket != "" {
		if err := m.uploadToS3(ctx, finalPath); err != nil {
			m.logger.WithError(err).Warn("Failed to upload to S3, keeping local backup")
		} else {
			m.logger.WithField("s3_bucket", m.config.S3Bucket).Info("Backup uploaded to S3")
		}
	}

	return finalPath, nil
}

// Restore restores a database from backup
func (m *Manager) Restore(ctx context.Context, backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	workingPath := backupPath

	// Decrypt if needed (check for .enc extension)
	if strings.HasSuffix(backupPath, ".enc") {
		if m.config.EncryptionKey == "" {
			return fmt.Errorf("backup is encrypted but no encryption key provided")
		}
		decryptedPath, err := m.decryptFile(ctx, backupPath)
		if err != nil {
			return fmt.Errorf("decrypting backup: %w", err)
		}
		defer os.Remove(decryptedPath)
		workingPath = decryptedPath
		m.logger.Info("Backup decrypted")
	}

	// Decompress if needed (check for .gz extension)
	if strings.HasSuffix(workingPath, ".gz") {
		decompressedPath, err := m.decompressFile(ctx, workingPath)
		if err != nil {
			return fmt.Errorf("decompressing backup: %w", err)
		}
		defer os.Remove(decompressedPath)
		workingPath = decompressedPath
		m.logger.Info("Backup decompressed")
	}

	// Restore using psql
	m.logger.WithField("backup_path", backupPath).Warn("Starting database restore - this will overwrite current data")

	cmd := exec.CommandContext(ctx, "psql", "-d", m.config.DatabaseURL, "-f", workingPath)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql restore failed: %w, output: %s", err, string(output))
	}

	m.logger.Info("Database restore completed successfully")
	return nil
}

// CleanupOldBackups removes backup files older than retention period
func (m *Manager) CleanupOldBackups(ctx context.Context) error {
	if m.config.RetentionDays <= 0 {
		m.logger.Debug("Retention days not set, skipping cleanup")
		return nil
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -m.config.RetentionDays)
	m.logger.WithField("cutoff_date", cutoff).Info("Cleaning up old backups")

	entries, err := os.ReadDir(m.config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading backup directory: %w", err)
	}

	var removed int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process backup files
		name := entry.Name()
		if !strings.HasPrefix(name, "backup_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			m.logger.WithError(err).WithField("file", name).Warn("Failed to get file info")
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(m.config.BackupDir, name)
			if err := os.Remove(path); err != nil {
				m.logger.WithError(err).WithField("file", name).Warn("Failed to remove old backup")
				continue
			}
			removed++
			m.logger.WithField("file", name).Debug("Removed old backup")
		}
	}

	m.logger.WithField("removed_count", removed).Info("Cleanup completed")
	return nil
}

// compressFile compresses a file using gzip
func (m *Manager) compressFile(ctx context.Context, path string) (string, error) {
	compressedPath := path + ".gz"

	cmd := exec.CommandContext(ctx, "gzip", "-c", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gzip failed: %w", err)
	}

	if err := os.WriteFile(compressedPath, output, 0600); err != nil {
		return "", fmt.Errorf("writing compressed file: %w", err)
	}

	return compressedPath, nil
}

// decompressFile decompresses a gzip file
func (m *Manager) decompressFile(ctx context.Context, path string) (string, error) {
	decompressedPath := strings.TrimSuffix(path, ".gz")

	cmd := exec.CommandContext(ctx, "gzip", "-d", "-c", path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gunzip failed: %w", err)
	}

	if err := os.WriteFile(decompressedPath, output, 0600); err != nil {
		return "", fmt.Errorf("writing decompressed file: %w", err)
	}

	return decompressedPath, nil
}

// encryptFile encrypts a file using openssl
func (m *Manager) encryptFile(ctx context.Context, path string) (string, error) {
	encryptedPath := path + ".enc"

	cmd := exec.CommandContext(ctx, "openssl", "enc", "-aes-256-cbc", "-salt",
		"-in", path,
		"-out", encryptedPath,
		"-pass", "pass:"+m.config.EncryptionKey)

	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(encryptedPath)
		return "", fmt.Errorf("openssl encrypt failed: %w, output: %s", err, string(output))
	}

	return encryptedPath, nil
}

// decryptFile decrypts a file using openssl
func (m *Manager) decryptFile(ctx context.Context, path string) (string, error) {
	decryptedPath := strings.TrimSuffix(path, ".enc")

	cmd := exec.CommandContext(ctx, "openssl", "enc", "-aes-256-cbc", "-d",
		"-in", path,
		"-out", decryptedPath,
		"-pass", "pass:"+m.config.EncryptionKey)

	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(decryptedPath)
		return "", fmt.Errorf("openssl decrypt failed: %w, output: %s", err, string(output))
	}

	return decryptedPath, nil
}

// uploadToS3 uploads a file to S3 using AWS CLI
func (m *Manager) uploadToS3(ctx context.Context, path string) error {
	if m.config.S3Bucket == "" {
		return nil
	}

	filename := filepath.Base(path)
	s3Path := fmt.Sprintf("s3://%s/%s", m.config.S3Bucket, filename)

	args := []string{"s3", "cp", path, s3Path}
	if m.config.S3Endpoint != "" {
		args = append(args, "--endpoint-url", m.config.S3Endpoint)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws s3 cp failed: %w, output: %s", err, string(output))
	}

	return nil
}

// VerifyBackup checks if a backup file is valid
func (m *Manager) VerifyBackup(ctx context.Context, backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Handle encrypted backups - we can only verify after decryption
	if strings.HasSuffix(backupPath, ".enc") {
		return fmt.Errorf("cannot verify encrypted backup directly, decrypt first")
	}

	// Verify gzip compressed files
	if strings.HasSuffix(backupPath, ".gz") {
		cmd := exec.CommandContext(ctx, "gzip", "-t", backupPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("gzip test failed: %w, output: %s", err, string(output))
		}
		return nil
	}

	// For plain SQL files, check if they're readable and non-empty
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("opening backup file: %w", err)
	}
	defer file.Close()

	// Check if file has content
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading backup file: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("backup file is empty")
	}

	// Check for SQL content indicators
	content := string(buf[:n])
	if !strings.Contains(content, "PostgreSQL") && !strings.Contains(content, "CREATE") &&
		!strings.Contains(content, "INSERT") && !strings.Contains(content, "--") {
		return fmt.Errorf("backup file does not appear to be a valid PostgreSQL dump")
	}

	return nil
}

// ListBackups returns a list of available backups
func (m *Manager) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(m.config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("reading backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "backup_") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backupType := "sql"
		if strings.HasSuffix(name, ".gz") {
			backupType = "compressed"
		}
		if strings.HasSuffix(name, ".enc") {
			backupType = "encrypted"
		}

		backups = append(backups, BackupInfo{
			Name:      name,
			Path:      filepath.Join(m.config.BackupDir, name),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Type:      backupType,
		})
	}

	return backups, nil
}

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Type      string    `json:"type"`
}
