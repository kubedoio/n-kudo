package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

type Store struct {
	mu   sync.Mutex
	path string
	data diskState
}

type diskState struct {
	Identity *Identity               `json:"identity,omitempty"`
	MicroVMs map[string]MicroVM      `json:"microvms"`
	Actions  map[string]ActionRecord `json:"actions"`
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	store := &Store{
		path: filepath.Join(dir, "edge-state.json"),
		data: diskState{
			MicroVMs: map[string]MicroVM{},
			Actions:  map[string]ActionRecord{},
		},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) SaveIdentity(identity Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	identity.SavedAt = time.Now().UTC()
	copied := identity
	s.data.Identity = &copied
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if vm.ID == "" {
		return errors.New("vm id required")
	}
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
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Store) GetActionRecord(actionID string) (ActionRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.data.Actions[actionID]
	return record, ok, nil
}

func (s *Store) PutActionRecord(record ActionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ActionID == "" {
		return errors.New("action id required")
	}
	record.UpdatedAt = time.Now().UTC()
	s.data.Actions[record.ActionID] = record
	return s.persistLocked()
}

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
	var loaded diskState
	if err := json.Unmarshal(b, &loaded); err != nil {
		return fmt.Errorf("decode state: %w", err)
	}
	if loaded.MicroVMs == nil {
		loaded.MicroVMs = map[string]MicroVM{}
	}
	if loaded.Actions == nil {
		loaded.Actions = map[string]ActionRecord{}
	}
	s.data = loaded
	return nil
}

func (s *Store) persistLocked() error {
	payload, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("commit state: %w", err)
	}
	return nil
}
