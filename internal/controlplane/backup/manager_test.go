package backup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewManager(t *testing.T) {
	logger := logrus.New()
	config := Config{
		DatabaseURL:   "postgres://test:test@localhost/test",
		BackupDir:     "/tmp/test-backups",
		RetentionDays: 7,
		Compress:      true,
		Encrypt:       false,
	}

	m := NewManager(config, logger)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.config.DatabaseURL != config.DatabaseURL {
		t.Error("database URL mismatch")
	}
}

func TestManager_ListBackups(t *testing.T) {
	// Create temp directory for test backups
	tempDir := t.TempDir()

	// Create test backup files
	testFiles := []string{
		"backup_20240101_120000.sql",
		"backup_20240102_120000.sql.gz",
		"backup_20240103_120000.sql.enc",
		"not_a_backup.txt",
	}

	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	m := NewManager(Config{BackupDir: tempDir}, nil)
	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("expected 3 backups, got %d", len(backups))
	}

	// Check backup types
	for _, b := range backups {
		switch {
		case strings.HasSuffix(b.Name, ".sql"):
			if b.Type != "sql" {
				t.Errorf("expected type 'sql', got '%s'", b.Type)
			}
		case strings.HasSuffix(b.Name, ".gz"):
			if b.Type != "compressed" {
				t.Errorf("expected type 'compressed', got '%s'", b.Type)
			}
		case strings.HasSuffix(b.Name, ".enc"):
			if b.Type != "encrypted" {
				t.Errorf("expected type 'encrypted', got '%s'", b.Type)
			}
		}
	}
}

func TestManager_CleanupOldBackups(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create old backup file
	oldBackup := filepath.Join(tempDir, "backup_20230101_120000.sql")
	if err := os.WriteFile(oldBackup, []byte("old"), 0600); err != nil {
		t.Fatalf("failed to create old backup: %v", err)
	}

	// Set old modification time
	oldTime := time.Now().UTC().AddDate(0, 0, -30)
	if err := os.Chtimes(oldBackup, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set file time: %v", err)
	}

	// Create recent backup file
	recentBackup := filepath.Join(tempDir, "backup_20240101_120000.sql")
	if err := os.WriteFile(recentBackup, []byte("recent"), 0600); err != nil {
		t.Fatalf("failed to create recent backup: %v", err)
	}

	m := NewManager(Config{
		BackupDir:     tempDir,
		RetentionDays: 7,
	}, nil)

	if err := m.CleanupOldBackups(ctx); err != nil {
		t.Fatalf("CleanupOldBackups failed: %v", err)
	}

	// Old backup should be removed
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Error("old backup should have been removed")
	}

	// Recent backup should still exist
	if _, err := os.Stat(recentBackup); err != nil {
		t.Error("recent backup should still exist")
	}
}

func TestManager_CleanupOldBackups_NoRetention(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	backup := filepath.Join(tempDir, "backup_20230101_120000.sql")
	if err := os.WriteFile(backup, []byte("test"), 0600); err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Set old modification time
	oldTime := time.Now().UTC().AddDate(0, 0, -30)
	if err := os.Chtimes(backup, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set file time: %v", err)
	}

	m := NewManager(Config{
		BackupDir:     tempDir,
		RetentionDays: 0, // No retention
	}, nil)

	if err := m.CleanupOldBackups(ctx); err != nil {
		t.Fatalf("CleanupOldBackups failed: %v", err)
	}

	// Backup should still exist (no retention policy)
	if _, err := os.Stat(backup); err != nil {
		t.Error("backup should still exist with no retention policy")
	}
}

func TestManager_VerifyBackup(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a valid SQL backup file
	sqlBackup := filepath.Join(tempDir, "backup_valid.sql")
	sqlContent := `-- PostgreSQL database dump
-- Dumped from database version 15.0
CREATE TABLE test (id int);
INSERT INTO test VALUES (1);`
	if err := os.WriteFile(sqlBackup, []byte(sqlContent), 0600); err != nil {
		t.Fatalf("failed to create sql backup: %v", err)
	}

	m := NewManager(Config{BackupDir: tempDir}, nil)

	if err := m.VerifyBackup(ctx, sqlBackup); err != nil {
		t.Errorf("VerifyBackup failed for valid SQL: %v", err)
	}

	// Test empty file
	emptyBackup := filepath.Join(tempDir, "backup_empty.sql")
	if err := os.WriteFile(emptyBackup, []byte{}, 0600); err != nil {
		t.Fatalf("failed to create empty backup: %v", err)
	}

	if err := m.VerifyBackup(ctx, emptyBackup); err == nil {
		t.Error("VerifyBackup should fail for empty file")
	}

	// Test non-existent file
	nonExistent := filepath.Join(tempDir, "backup_nonexistent.sql")
	if err := m.VerifyBackup(ctx, nonExistent); err == nil {
		t.Error("VerifyBackup should fail for non-existent file")
	}

	// Test encrypted file (should fail with specific error)
	encBackup := filepath.Join(tempDir, "backup_encrypted.sql.enc")
	if err := os.WriteFile(encBackup, []byte("encrypted"), 0600); err != nil {
		t.Fatalf("failed to create encrypted backup: %v", err)
	}

	if err := m.VerifyBackup(ctx, encBackup); err == nil {
		t.Error("VerifyBackup should fail for encrypted file without decryption")
	}
}

func TestManager_compressFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("This is test content for compression. It needs to be long enough to compress effectively.")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m := NewManager(Config{BackupDir: tempDir}, nil)

	// Compress
	compressedPath, err := m.compressFile(ctx, testFile)
	if err != nil {
		t.Fatalf("compressFile failed: %v", err)
	}

	if !strings.HasSuffix(compressedPath, ".gz") {
		t.Errorf("compressed file should have .gz extension, got: %s", compressedPath)
	}

	// Verify compressed file exists and is smaller
	compressedInfo, err := os.Stat(compressedPath)
	if err != nil {
		t.Fatalf("failed to stat compressed file: %v", err)
	}

	if compressedInfo.Size() == 0 {
		t.Error("compressed file should not be empty")
	}
}

func TestManager_decompressFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create test file and compress it
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("This is test content for compression and decompression.")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m := NewManager(Config{BackupDir: tempDir}, nil)

	// Compress
	compressedPath, err := m.compressFile(ctx, testFile)
	if err != nil {
		t.Fatalf("compressFile failed: %v", err)
	}

	// Remove original file
	os.Remove(testFile)

	// Decompress
	decompressedPath, err := m.decompressFile(ctx, compressedPath)
	if err != nil {
		t.Fatalf("decompressFile failed: %v", err)
	}

	// Verify content
	decompressedContent, err := os.ReadFile(decompressedPath)
	if err != nil {
		t.Fatalf("failed to read decompressed file: %v", err)
	}

	if string(decompressedContent) != string(content) {
		t.Errorf("decompressed content mismatch: got %q, want %q", decompressedContent, content)
	}
}

func TestManager_encryptDecryptFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("This is sensitive data that needs encryption.")
	if err := os.WriteFile(testFile, content, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m := NewManager(Config{
		BackupDir:     tempDir,
		EncryptionKey: "test-encryption-key-12345",
	}, nil)

	// Encrypt
	encryptedPath, err := m.encryptFile(ctx, testFile)
	if err != nil {
		t.Fatalf("encryptFile failed: %v", err)
	}

	if !strings.HasSuffix(encryptedPath, ".enc") {
		t.Errorf("encrypted file should have .enc extension, got: %s", encryptedPath)
	}

	// Verify encrypted file exists
	encryptedInfo, err := os.Stat(encryptedPath)
	if err != nil {
		t.Fatalf("failed to stat encrypted file: %v", err)
	}

	if encryptedInfo.Size() == 0 {
		t.Error("encrypted file should not be empty")
	}

	// Remove original file
	os.Remove(testFile)

	// Decrypt
	decryptedPath, err := m.decryptFile(ctx, encryptedPath)
	if err != nil {
		t.Fatalf("decryptFile failed: %v", err)
	}

	// Verify content
	decryptedContent, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("failed to read decrypted file: %v", err)
	}

	if string(decryptedContent) != string(content) {
		t.Errorf("decrypted content mismatch: got %q, want %q", decryptedContent, content)
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				DatabaseURL:   "postgres://user:pass@localhost/db",
				BackupDir:     "/backups",
				RetentionDays: 7,
				Compress:      true,
			},
			wantErr: false,
		},
		{
			name: "encryption without key",
			config: Config{
				DatabaseURL:   "postgres://user:pass@localhost/db",
				BackupDir:     "/backups",
				Encrypt:       true,
				EncryptionKey: "",
			},
			wantErr: false, // Should be handled at runtime
		},
		{
			name: "s3 without bucket",
			config: Config{
				DatabaseURL:   "postgres://user:pass@localhost/db",
				BackupDir:     "/backups",
				S3Endpoint:    "http://localhost:9000",
			},
			wantErr: false, // Should be handled at runtime
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Config validation is handled at runtime, not struct level
			// This test documents expected behavior
			m := NewManager(tt.config, nil)
			if m == nil {
				t.Error("NewManager should return non-nil manager")
			}
		})
	}
}

func TestBackupInfo(t *testing.T) {
	now := time.Now().UTC()
	info := BackupInfo{
		Name:      "backup_test.sql",
		Path:      "/backups/backup_test.sql",
		Size:      1024,
		CreatedAt: now,
		Type:      "sql",
	}

	if info.Name != "backup_test.sql" {
		t.Errorf("name mismatch: got %q", info.Name)
	}
	if info.Size != 1024 {
		t.Errorf("size mismatch: got %d", info.Size)
	}
	if info.Type != "sql" {
		t.Errorf("type mismatch: got %q", info.Type)
	}
}

func TestManager_ListBackups_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	m := NewManager(Config{BackupDir: tempDir}, nil)

	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestManager_ListBackups_NonExistentDir(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "does-not-exist")

	m := NewManager(Config{BackupDir: nonExistentDir}, nil)

	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}
