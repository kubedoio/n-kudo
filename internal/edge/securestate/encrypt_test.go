package securestate

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubedoio/n-kudo/internal/edge/state"
)

func TestDeriveKey_FromEnvRaw(t *testing.T) {
	// Use a valid 32-byte string (printable ASCII to avoid null byte issues)
	key := "this-is-a-test-key-with-32-bytes"
	if len(key) != KeySize {
		t.Fatalf("Test key must be %d bytes, got %d", KeySize, len(key))
	}

	os.Setenv("NKUDO_STATE_KEY", key)
	defer os.Unsetenv("NKUDO_STATE_KEY")

	derived, err := DeriveKey()
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	if !bytes.Equal(derived, []byte(key)) {
		t.Errorf("Expected key to match, got %v", derived)
	}
}

func TestDeriveKey_FromEnvBase64(t *testing.T) {
	// Generate a 32-byte key and encode as base64
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	encoded := base64.StdEncoding.EncodeToString(key)

	os.Setenv("NKUDO_STATE_KEY", encoded)
	defer os.Unsetenv("NKUDO_STATE_KEY")

	derived, err := DeriveKey()
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	if !bytes.Equal(derived, key) {
		t.Errorf("Expected key to match, got %v", derived)
	}
}

func TestDeriveKey_NoEnv(t *testing.T) {
	os.Unsetenv("NKUDO_STATE_KEY")

	derived, err := DeriveKey()
	if err != nil {
		t.Fatalf("DeriveKey should not error when env not set: %v", err)
	}
	if derived != nil {
		t.Error("Expected nil key when env not set")
	}
}

func TestDeriveKey_InvalidKey(t *testing.T) {
	os.Setenv("NKUDO_STATE_KEY", "too-short")
	defer os.Unsetenv("NKUDO_STATE_KEY")

	_, err := DeriveKey()
	if err == nil {
		t.Error("Expected error for invalid key")
	}
}

func TestParseKey_Raw32Bytes(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	parsed, err := parseKey(string(key))
	if err != nil {
		t.Fatalf("parseKey failed: %v", err)
	}
	if !bytes.Equal(parsed, key) {
		t.Error("Parsed key does not match")
	}
}

func TestParseKey_Base64(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	encoded := base64.StdEncoding.EncodeToString(key)

	parsed, err := parseKey(encoded)
	if err != nil {
		t.Fatalf("parseKey failed: %v", err)
	}
	if !bytes.Equal(parsed, key) {
		t.Error("Parsed key does not match")
	}
}

func TestParseKey_InvalidLength(t *testing.T) {
	_, err := parseKey("short")
	if err == nil {
		t.Error("Expected error for short key")
	}

	_, err = parseKey(string(make([]byte, 64)))
	if err == nil {
		t.Error("Expected error for long key")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	plaintext := []byte("Hello, World! This is a test message.")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Check format
	if len(ciphertext) < 1+NonceSize {
		t.Error("Ciphertext too short")
	}
	if ciphertext[0] != VersionByte {
		t.Errorf("Expected version byte 0x%02x, got 0x%02x", VersionByte, ciphertext[0])
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted text does not match: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_InvalidKey(t *testing.T) {
	_, err := Encrypt([]byte("short"), []byte("test"))
	if err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}
}

func TestDecrypt_InvalidKey(t *testing.T) {
	_, err := Decrypt([]byte("short"), []byte("test"))
	if err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey, got %v", err)
	}
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key, _ := GenerateKey()

	// Too short
	_, err := Decrypt(key, []byte{VersionByte})
	if !errors.Is(err, ErrInvalidCiphertext) {
		t.Errorf("Expected ErrInvalidCiphertext, got %v", err)
	}

	// Wrong version - check that it contains ErrVersionMismatch
	_, err = Decrypt(key, []byte{0x99, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if !errors.Is(err, ErrVersionMismatch) {
		t.Errorf("Expected ErrVersionMismatch, got %v", err)
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key, _ := GenerateKey()
	plaintext := []byte("secret message")

	ciphertext, _ := Encrypt(key, plaintext)

	// Tamper with the ciphertext (after nonce)
	ciphertext[1+NonceSize+5] ^= 0xFF

	_, err := Decrypt(key, ciphertext)
	if err == nil {
		t.Error("Expected error for tampered ciphertext")
	}
}

func TestIsEncrypted(t *testing.T) {
	key, _ := GenerateKey()
	ciphertext, _ := Encrypt(key, []byte("test"))

	if !IsEncrypted(ciphertext) {
		t.Error("IsEncrypted should return true for encrypted data")
	}

	if IsEncrypted([]byte("not encrypted")) {
		t.Error("IsEncrypted should return false for plaintext")
	}

	if IsEncrypted([]byte{}) {
		t.Error("IsEncrypted should return false for empty data")
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if len(key) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key))
	}

	// Generate another key to ensure randomness
	key2, _ := GenerateKey()
	if bytes.Equal(key, key2) {
		t.Error("Generated keys should be different")
	}
}

func TestGenerateKeyBase64(t *testing.T) {
	encoded, err := GenerateKeyBase64()
	if err != nil {
		t.Fatalf("GenerateKeyBase64 failed: %v", err)
	}

	// Decode and verify length
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64 key: %v", err)
	}
	if len(decoded) != KeySize {
		t.Errorf("Expected decoded key size %d, got %d", KeySize, len(decoded))
	}
}

func TestStore_OpenEncrypted(t *testing.T) {
	tmpDir := t.TempDir()
	key, _ := GenerateKey()

	store, err := OpenWithKey(tmpDir, key)
	if err != nil {
		t.Fatalf("OpenWithKey failed: %v", err)
	}
	defer store.Close()

	if !store.IsEncrypted() {
		t.Error("Expected store to be encrypted")
	}

	// Save some data
	identity := state.Identity{
		TenantID:     "tenant-1",
		SiteID:       "site-1",
		HostID:       "host-1",
		AgentID:      "agent-1",
		RefreshToken: "secret-token",
	}
	if err := store.SaveIdentity(identity); err != nil {
		t.Fatalf("SaveIdentity failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(tmpDir, "edge-state-encrypted.json")); err != nil {
		t.Error("Encrypted state file not created")
	}

	// Reopen and verify data
	store.Close()
	store2, err := OpenWithKey(tmpDir, key)
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer store2.Close()

	loaded, err := store2.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity failed: %v", err)
	}
	if loaded.AgentID != identity.AgentID {
		t.Errorf("Expected AgentID %s, got %s", identity.AgentID, loaded.AgentID)
	}
	if loaded.RefreshToken != identity.RefreshToken {
		t.Errorf("Expected RefreshToken %s, got %s", identity.RefreshToken, loaded.RefreshToken)
	}
}

func TestStore_OpenUnencrypted(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := OpenWithKey(tmpDir, nil)
	if err != nil {
		t.Fatalf("OpenWithKey with nil key failed: %v", err)
	}
	defer store.Close()

	if store.IsEncrypted() {
		t.Error("Expected store to be unencrypted")
	}

	// Save some data
	identity := state.Identity{
		TenantID: "tenant-1",
		SiteID:   "site-1",
		AgentID:  "agent-1",
	}
	if err := store.SaveIdentity(identity); err != nil {
		t.Fatalf("SaveIdentity failed: %v", err)
	}

	// Verify file was created with unencrypted name
	if _, err := os.Stat(filepath.Join(tmpDir, "edge-state.json")); err != nil {
		t.Error("Unencrypted state file not created")
	}

	// Verify content is plaintext
	content, _ := os.ReadFile(filepath.Join(tmpDir, "edge-state.json"))
	if IsEncrypted(content) {
		t.Error("State file should not be encrypted")
	}
}

func TestStore_EncryptedFileRequiresKey(t *testing.T) {
	tmpDir := t.TempDir()
	key, _ := GenerateKey()

	// Create encrypted store
	store, _ := OpenWithKey(tmpDir, key)
	store.SaveIdentity(state.Identity{AgentID: "agent-1"})
	store.Close()

	// Try to open without key using Open() which checks for encrypted file
	os.Unsetenv("NKUDO_STATE_KEY")
	_, err := Open(tmpDir)
	if err == nil {
		t.Error("Expected error when opening encrypted file without key")
	}
}

func TestStore_MicroVMOperations(t *testing.T) {
	tmpDir := t.TempDir()
	key, _ := GenerateKey()

	store, _ := OpenWithKey(tmpDir, key)
	defer store.Close()

	vm := state.MicroVM{
		ID:         "vm-1",
		Name:       "test-vm",
		KernelPath: "/path/to/kernel",
		Status:     "RUNNING",
	}

	// Test Upsert
	if err := store.UpsertMicroVM(vm); err != nil {
		t.Fatalf("UpsertMicroVM failed: %v", err)
	}

	// Test Get
	loaded, found, err := store.GetMicroVM("vm-1")
	if err != nil {
		t.Fatalf("GetMicroVM failed: %v", err)
	}
	if !found {
		t.Error("Expected VM to be found")
	}
	if loaded.Name != vm.Name {
		t.Errorf("Expected Name %s, got %s", vm.Name, loaded.Name)
	}

	// Test List
	vms, err := store.ListMicroVMs()
	if err != nil {
		t.Fatalf("ListMicroVMs failed: %v", err)
	}
	if len(vms) != 1 {
		t.Errorf("Expected 1 VM, got %d", len(vms))
	}

	// Test Delete
	if err := store.DeleteMicroVM("vm-1"); err != nil {
		t.Fatalf("DeleteMicroVM failed: %v", err)
	}

	_, found, _ = store.GetMicroVM("vm-1")
	if found {
		t.Error("Expected VM to be deleted")
	}
}

func TestStore_ActionRecordOperations(t *testing.T) {
	tmpDir := t.TempDir()
	key, _ := GenerateKey()

	store, _ := OpenWithKey(tmpDir, key)
	defer store.Close()

	record := state.ActionRecord{
		ActionID:    "action-1",
		ExecutionID: "exec-1",
		OK:          true,
		Message:     "success",
	}

	// Test Put
	if err := store.PutActionRecord(record); err != nil {
		t.Fatalf("PutActionRecord failed: %v", err)
	}

	// Test Get
	loaded, found, err := store.GetActionRecord("action-1")
	if err != nil {
		t.Fatalf("GetActionRecord failed: %v", err)
	}
	if !found {
		t.Error("Expected record to be found")
	}
	if loaded.Message != record.Message {
		t.Errorf("Expected Message %s, got %s", record.Message, loaded.Message)
	}
}

func TestStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	key, _ := GenerateKey()

	// Create store and add data
	store1, _ := OpenWithKey(tmpDir, key)
	store1.SaveIdentity(state.Identity{AgentID: "agent-1", TenantID: "tenant-1"})
	store1.UpsertMicroVM(state.MicroVM{ID: "vm-1", Name: "test-vm"})
	store1.PutActionRecord(state.ActionRecord{ActionID: "action-1", OK: true})
	store1.Close()

	// Reopen and verify
	store2, _ := OpenWithKey(tmpDir, key)
	defer store2.Close()

	identity, _ := store2.LoadIdentity()
	if identity.AgentID != "agent-1" {
		t.Error("Identity not persisted correctly")
	}

	vm, found, _ := store2.GetMicroVM("vm-1")
	if !found || vm.Name != "test-vm" {
		t.Error("MicroVM not persisted correctly")
	}

	record, found, _ := store2.GetActionRecord("action-1")
	if !found || !record.OK {
		t.Error("ActionRecord not persisted correctly")
	}
}

func TestMigrateFromUnencrypted(t *testing.T) {
	tmpDir := t.TempDir()
	oldPath := filepath.Join(tmpDir, "edge-state.json")

	// Create old unencrypted state file
	oldData := `{"identity":{"agent_id":"agent-1","tenant_id":"tenant-1"},"microvms":{"vm-1":{"id":"vm-1","name":"test-vm"}},"actions":{}}`
	os.WriteFile(oldPath, []byte(oldData), 0o600)

	// Create encrypted store
	key, _ := GenerateKey()
	encryptedStore, _ := OpenWithKey(tmpDir, key)
	defer encryptedStore.Close()

	// Migrate
	if err := MigrateFromUnencrypted(encryptedStore, oldPath); err != nil {
		t.Fatalf("MigrateFromUnencrypted failed: %v", err)
	}

	// Verify data was migrated
	identity, _ := encryptedStore.LoadIdentity()
	if identity.AgentID != "agent-1" {
		t.Error("Identity not migrated correctly")
	}

	vm, found, _ := encryptedStore.GetMicroVM("vm-1")
	if !found || vm.Name != "test-vm" {
		t.Error("MicroVM not migrated correctly")
	}
}

func TestMigrateFromUnencrypted_AlreadyEncrypted(t *testing.T) {
	tmpDir := t.TempDir()
	key, _ := GenerateKey()

	// Create an encrypted file
	encryptedData, _ := Encrypt(key, []byte(`{"identity":{"agent_id":"test"}}`))
	oldPath := filepath.Join(tmpDir, "edge-state.json")
	os.WriteFile(oldPath, encryptedData, 0o600)

	encryptedStore, _ := OpenWithKey(tmpDir, key)
	defer encryptedStore.Close()

	err := MigrateFromUnencrypted(encryptedStore, oldPath)
	if err == nil {
		t.Error("Expected error when migrating already encrypted file")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateKey()
	plaintext := make([]byte, 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Encrypt(key, plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateKey()
	plaintext := make([]byte, 1024) // 1KB
	ciphertext, _ := Encrypt(key, plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decrypt(key, ciphertext)
		if err != nil {
			b.Fatal(err)
		}
	}
}
