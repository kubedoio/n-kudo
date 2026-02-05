package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Identity struct {
	TenantID     string    `json:"tenant_id"`
	SiteID       string    `json:"site_id"`
	HostID       string    `json:"host_id"`
	AgentID      string    `json:"agent_id"`
	RefreshToken string    `json:"refresh_token"`
	SavedAt      time.Time `json:"saved_at"`
}

type MicroVM struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	KernelPath string    `json:"kernel_path"`
	RootfsPath string    `json:"rootfs_path"`
	TapIface   string    `json:"tap_iface"`
	CHPID      int       `json:"ch_pid"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ActionRecord struct {
	ActionID    string    `json:"action_id"`
	ExecutionID string    `json:"execution_id"`
	OK          bool      `json:"ok"`
	ErrorCode   string    `json:"error_code,omitempty"`
	Message     string    `json:"message"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type dbFile struct {
	Identity *Identity               `json:"identity,omitempty"`
	MicroVMs map[string]MicroVM      `json:"microvms"`
	Actions  map[string]ActionRecord `json:"actions"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data dbFile
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	path := filepath.Join(dir, "edge-state.json")
	s := &Store{path: path}
	s.data.MicroVMs = map[string]MicroVM{}
	s.data.Actions = map[string]ActionRecord{}

	if _, err := os.Stat(path); err == nil {
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read state: %w", readErr)
		}
		if len(b) > 0 {
			if err := json.Unmarshal(b, &s.data); err != nil {
				return nil, fmt.Errorf("decode state: %w", err)
			}
			if s.data.MicroVMs == nil {
				s.data.MicroVMs = map[string]MicroVM{}
			}
			if s.data.Actions == nil {
				s.data.Actions = map[string]ActionRecord{}
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else {
		if err := s.persistLocked(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) SaveIdentity(identity Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	identity.SavedAt = time.Now().UTC()
	s.data.Identity = &identity
	return s.persistLocked()
}

func (s *Store) LoadIdentity() (Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Identity == nil {
		return Identity{}, errors.New("identity not found")
	}
	return *s.data.Identity, nil
}

func (s *Store) UpsertMicroVM(vm MicroVM) error {
	if vm.ID == "" {
		return errors.New("vm id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	vm.UpdatedAt = time.Now().UTC()
	s.data.MicroVMs[vm.ID] = vm
	return s.persistLocked()
}

func (s *Store) GetMicroVM(vmID string) (MicroVM, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vm, ok := s.data.MicroVMs[vmID]
	return vm, ok, nil
}

func (s *Store) DeleteMicroVM(vmID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.MicroVMs, vmID)
	return s.persistLocked()
}

func (s *Store) ListMicroVMs() ([]MicroVM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]MicroVM, 0, len(s.data.MicroVMs))
	for _, vm := range s.data.MicroVMs {
		out = append(out, vm)
	}
	return out, nil
}

func (s *Store) GetActionRecord(actionID string) (ActionRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.data.Actions[actionID]
	return rec, ok, nil
}

func (s *Store) PutActionRecord(record ActionRecord) error {
	if record.ActionID == "" {
		return errors.New("action id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record.UpdatedAt = time.Now().UTC()
	s.data.Actions[record.ActionID] = record
	return s.persistLocked()
}

func (s *Store) persistLocked() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
