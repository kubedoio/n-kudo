package mocks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockNetBirdServer simulates the NetBird management API
type MockNetBirdServer struct {
	Server *httptest.Server
	mu     sync.RWMutex

	// Registered peers
	Peers map[string]*MockPeer

	// Setup keys
	SetupKeys map[string]*MockSetupKey

	// Call tracking
	PeerRegisterCalls int
	StatusCalls       int
}

type MockPeer struct {
	ID        string
	Name      string
	SetupKey  string
	Connected bool
	IP        string
}

type MockSetupKey struct {
	ID        string
	Name      string
	Revoked   bool
	ExpiresAt string
}

func NewMockNetBirdServer() *MockNetBirdServer {
	m := &MockNetBirdServer{
		Peers:     make(map[string]*MockPeer),
		SetupKeys: make(map[string]*MockSetupKey),
	}

	mux := http.NewServeMux()

	// Peer registration endpoint
	mux.HandleFunc("/api/peers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			m.mu.Lock()
			m.PeerRegisterCalls++
			m.mu.Unlock()

			var req struct {
				Name     string `json:"name"`
				SetupKey string `json:"setup_key"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			m.mu.Lock()
			peer := &MockPeer{
				ID:        "peer-" + req.Name,
				Name:      req.Name,
				SetupKey:  req.SetupKey,
				Connected: true,
				IP:        "100.64.0.1",
			}
			m.Peers[peer.ID] = peer
			m.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(peer)
			return
		}

		if r.Method == http.MethodGet {
			m.mu.RLock()
			defer m.mu.RUnlock()

			peers := make([]*MockPeer, 0, len(m.Peers))
			for _, p := range m.Peers {
				peers = append(peers, p)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(peers)
			return
		}
	})

	// Peer status endpoint
	mux.HandleFunc("/api/peers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			m.mu.RLock()
			m.StatusCalls++
			m.mu.RUnlock()

			peerID := r.URL.Path[len("/api/peers/"):]

			m.mu.RLock()
			peer, exists := m.Peers[peerID]
			m.mu.RUnlock()

			if !exists {
				http.Error(w, "peer not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(peer)
			return
		}
	})

	// Setup key validation
	mux.HandleFunc("/api/setup-keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			m.mu.RLock()
			defer m.mu.RUnlock()

			keys := make([]*MockSetupKey, 0, len(m.SetupKeys))
			for _, k := range m.SetupKeys {
				keys = append(keys, k)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(keys)
			return
		}
	})

	m.Server = httptest.NewServer(mux)
	return m
}

func (m *MockNetBirdServer) AddSetupKey(id, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SetupKeys[id] = &MockSetupKey{
		ID:   id,
		Name: name,
	}
}

func (m *MockNetBirdServer) RevokeSetupKey(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if key, exists := m.SetupKeys[id]; exists {
		key.Revoked = true
	}
}

func (m *MockNetBirdServer) Close() {
	m.Server.Close()
}

func (m *MockNetBirdServer) URL() string {
	return m.Server.URL
}

func (m *MockNetBirdServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Peers = make(map[string]*MockPeer)
	m.SetupKeys = make(map[string]*MockSetupKey)
	m.PeerRegisterCalls = 0
	m.StatusCalls = 0
}
