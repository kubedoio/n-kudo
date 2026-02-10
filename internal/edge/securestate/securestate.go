package securestate

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/state"
)

// Store wraps the standard state.Store with optional encryption at rest.
// It provides transparent encryption/decryption for sensitive state files.
type Store struct {
	mu        sync.Mutex
	path      string
	key       []byte
	encrypted bool
	data      diskState
}

type diskState struct {
	Identity *state.Identity               `json:"identity,omitempty"`
	MicroVMs map[string]state.MicroVM      `json:"microvms"`
	Actions  map[string]state.ActionRecord `json:"actions"`
}

// Open opens or creates a secure state store at the given directory.
// If NKUDO_STATE_KEY is set, encryption will be enabled.
// If no key is available, it falls back to unencrypted mode for backward compatibility.
func Open(dir string) (*Store, error) {
	// Try to derive encryption key
	key, err := DeriveKey()
	if err != nil {
		return nil, fmt.Errorf("derive encryption key: %w", err)
	}

	store := &Store{
		path: filepath.Join(dir, "edge-state-encrypted.json"),
		key:  key,
		data: diskState{
			MicroVMs: map[string]state.MicroVM{},
			Actions:  map[string]state.ActionRecord{},
		},
	}

	if key != nil {
		store.encrypted = true
		log.Println("[securestate] Encryption enabled for state store")
	} else {
		// Check if there's an existing encrypted file - we need the key
		encryptedPath := filepath.Join(dir, "edge-state-encrypted.json")
		if _, err := os.Stat(encryptedPath); err == nil {
			return nil, fmt.Errorf("encrypted state file exists at %s but NKUDO_STATE_KEY is not set", encryptedPath)
		}

		// Fall back to unencrypted path
		store.path = filepath.Join(dir, "edge-state.json")
		log.Println("[securestate] Encryption disabled - state stored in plaintext (set NKUDO_STATE_KEY to enable)")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// OpenWithKey opens a secure state store with an explicit key.
// This is useful for testing or when the key is obtained from an external source.
func OpenWithKey(dir string, key []byte) (*Store, error) {
	if key != nil && len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	store := &Store{
		path: filepath.Join(dir, "edge-state-encrypted.json"),
		key:  key,
		data: diskState{
			MicroVMs: map[string]state.MicroVM{},
			Actions:  map[string]state.ActionRecord{},
		},
	}

	if key != nil {
		store.encrypted = true
	} else {
		store.path = filepath.Join(dir, "edge-state.json")
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// Close closes the store (currently a no-op for compatibility)
func (s *Store) Close() error {
	return nil
}

// IsEncrypted returns true if encryption is enabled
func (s *Store) IsEncrypted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encrypted
}

// SaveIdentity saves the identity to the store
func (s *Store) SaveIdentity(identity state.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	identity.SavedAt = time.Now().UTC()
	copied := identity
	s.data.Identity = &copied
	return s.persistLocked()
}

// LoadIdentity loads the identity from the store
func (s *Store) LoadIdentity() (state.Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.Identity == nil {
		return state.Identity{}, errors.New("identity not found")
	}
	return *s.data.Identity, nil
}

// UpsertMicroVM inserts or updates a MicroVM record
func (s *Store) UpsertMicroVM(vm state.MicroVM) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if vm.ID == "" {
		return errors.New("vm id required")
	}
	vm.UpdatedAt = time.Now().UTC()
	s.data.MicroVMs[vm.ID] = vm
	return s.persistLocked()
}

// GetMicroVM retrieves a MicroVM by ID
func (s *Store) GetMicroVM(vmID string) (state.MicroVM, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	vm, ok := s.data.MicroVMs[vmID]
	return vm, ok, nil
}

// DeleteMicroVM removes a MicroVM from the store
func (s *Store) DeleteMicroVM(vmID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data.MicroVMs, vmID)
	return s.persistLocked()
}

// ListMicroVMs returns all MicroVMs sorted by ID
func (s *Store) ListMicroVMs() ([]state.MicroVM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]state.MicroVM, 0, len(s.data.MicroVMs))
	for _, vm := range s.data.MicroVMs {
		out = append(out, vm)
	}
	// Simple sort by ID
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i].ID > out[j].ID {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// GetActionRecord retrieves an action record by ID
func (s *Store) GetActionRecord(actionID string) (state.ActionRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data.Actions[actionID]
	return record, ok, nil
}

// PutActionRecord inserts or updates an action record
func (s *Store) PutActionRecord(record state.ActionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ActionID == "" {
		return errors.New("action id required")
	}
	record.UpdatedAt = time.Now().UTC()
	s.data.Actions[record.ActionID] = record
	return s.persistLocked()
}

// load reads the state from disk
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	if len(b) == 0 {
		return nil
	}

	var plaintext []byte
	if s.encrypted {
		// Check if file looks encrypted
		if !IsEncrypted(b) {
			return fmt.Errorf("state file %s appears to be unencrypted but encryption key is set", s.path)
		}
		plaintext, err = Decrypt(s.key, b)
		if err != nil {
			return fmt.Errorf("decrypt state: %w", err)
		}
	} else {
		plaintext = b
	}

	var loaded diskState
	if err := json.Unmarshal(plaintext, &loaded); err != nil {
		return fmt.Errorf("decode state: %w", err)
	}
	if loaded.MicroVMs == nil {
		loaded.MicroVMs = map[string]state.MicroVM{}
	}
	if loaded.Actions == nil {
		loaded.Actions = map[string]state.ActionRecord{}
	}
	s.data = loaded
	return nil
}

// persistLocked writes the state to disk (caller must hold lock)
func (s *Store) persistLocked() error {
	payload, err := json.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	var dataToWrite []byte
	if s.encrypted {
		dataToWrite, err = Encrypt(s.key, payload)
		if err != nil {
			return fmt.Errorf("encrypt state: %w", err)
		}
	} else {
		dataToWrite = payload
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, dataToWrite, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("commit state: %w", err)
	}
	return nil
}

// MigrateFromUnencrypted migrates data from an unencrypted state store.
// This is useful for upgrading existing installations to use encryption.
func MigrateFromUnencrypted(encryptedStore *Store, oldPath string) error {
	if !encryptedStore.encrypted {
		return errors.New("target store is not encrypted")
	}

	b, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("read old state: %w", err)
	}

	if len(b) == 0 {
		return nil
	}

	// Check if already encrypted
	if IsEncrypted(b) {
		return errors.New("old state file is already encrypted")
	}

	var loaded diskState
	if err := json.Unmarshal(b, &loaded); err != nil {
		return fmt.Errorf("decode old state: %w", err)
	}

	encryptedStore.mu.Lock()
	defer encryptedStore.mu.Unlock()

	encryptedStore.data = loaded
	if loaded.MicroVMs == nil {
		encryptedStore.data.MicroVMs = map[string]state.MicroVM{}
	}
	if loaded.Actions == nil {
		encryptedStore.data.Actions = map[string]state.ActionRecord{}
	}

	return encryptedStore.persistLocked()
}
