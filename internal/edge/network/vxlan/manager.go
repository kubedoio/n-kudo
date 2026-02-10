package vxlan

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager manages VXLAN tunnels on the edge node
type Manager struct {
	tunnels map[string]*VXLANTunnel // key: VTEP name
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new VXLAN manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		tunnels: make(map[string]*VXLANTunnel),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Stop stops the VXLAN manager and cleans up all tunnels
func (m *Manager) Stop() error {
	m.cancel()
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var lastErr error
	// Clean up all tunnels
	for vtepName := range m.tunnels {
		if err := m.teardownTunnelUnsafe(vtepName); err != nil {
			lastErr = err
			log.Printf("[vxlan] error tearing down tunnel %s during stop: %v", vtepName, err)
		}
	}
	
	return lastErr
}

// SetupTunnel creates and configures a new VXLAN tunnel
func (m *Manager) SetupTunnel(ctx context.Context, config VXLANConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if tunnel already exists
	if _, exists := m.tunnels[config.VTEPName]; exists {
		return fmt.Errorf("tunnel for VTEP %s already exists", config.VTEPName)
	}

	tunnel := &VXLANTunnel{
		Config:    config,
		Status:    TunnelStatusCreating,
		CreatedAt: time.Now().UTC(),
	}
	m.tunnels[config.VTEPName] = tunnel

	// Create bridge if specified
	bridgeName := config.VTEPName + "-br"
	tunnel.BridgeName = bridgeName

	if err := CreateBridge(bridgeName, config.MTU); err != nil {
		if _, ok := err.(*BridgeExistsError); !ok {
			tunnel.Status = TunnelStatusFailed
			tunnel.LastError = fmt.Sprintf("failed to create bridge: %v", err)
			return fmt.Errorf("failed to create bridge: %w", err)
		}
		// Bridge exists, continue
	}

	// Create VTEP
	if err := CreateVTEP(config); err != nil {
		tunnel.Status = TunnelStatusFailed
		tunnel.LastError = fmt.Sprintf("failed to create VTEP: %v", err)
		// Clean up bridge if we created it
		_ = DeleteBridge(bridgeName)
		return fmt.Errorf("failed to create VTEP: %w", err)
	}

	// Attach VTEP to bridge
	if err := AttachToBridge(config.VTEPName, bridgeName); err != nil {
		tunnel.Status = TunnelStatusFailed
		tunnel.LastError = fmt.Sprintf("failed to attach VTEP to bridge: %v", err)
		// Clean up
		_ = DeleteVTEP(config.VTEPName)
		_ = DeleteBridge(bridgeName)
		return fmt.Errorf("failed to attach VTEP to bridge: %w", err)
	}

	tunnel.Status = TunnelStatusActive
	log.Printf("[vxlan] tunnel %s created and attached to bridge %s", config.VTEPName, bridgeName)

	return nil
}

// TeardownTunnel removes a VXLAN tunnel and cleans up resources
func (m *Manager) TeardownTunnel(vni int) error {
	vtepName := GenerateVTEPName(vni)

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.teardownTunnelUnsafe(vtepName)
}

// teardownTunnelUnsafe removes a tunnel without locking (caller must hold lock)
func (m *Manager) teardownTunnelUnsafe(vtepName string) error {
	tunnel, exists := m.tunnels[vtepName]
	if !exists {
		return fmt.Errorf("tunnel for VTEP %s not found", vtepName)
	}

	tunnel.Status = TunnelStatusTearingDown

	// Detach VTEP from bridge first
	if err := DetachFromBridge(vtepName); err != nil {
		log.Printf("[vxlan] warning: failed to detach VTEP from bridge: %v", err)
		// Continue with cleanup
	}

	// Delete VTEP
	if err := DeleteVTEP(vtepName); err != nil {
		if _, ok := err.(*VTEPNotFoundError); !ok {
			tunnel.LastError = fmt.Sprintf("failed to delete VTEP: %v", err)
			return fmt.Errorf("failed to delete VTEP: %w", err)
		}
	}

	// Delete bridge
	if tunnel.BridgeName != "" {
		if err := DeleteBridge(tunnel.BridgeName); err != nil {
			if _, ok := err.(*BridgeNotFoundError); !ok {
				log.Printf("[vxlan] warning: failed to delete bridge: %v", err)
				// Continue with cleanup
			}
		}
	}

	tunnel.Status = TunnelStatusDestroyed
	delete(m.tunnels, vtepName)

	log.Printf("[vxlan] tunnel %s torn down", vtepName)

	return nil
}

// GetTunnel returns a tunnel by VNI
func (m *Manager) GetTunnel(vni int) (*VXLANTunnel, error) {
	vtepName := GenerateVTEPName(vni)

	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[vtepName]
	if !exists {
		return nil, fmt.Errorf("tunnel for VNI %d not found", vni)
	}

	// Return a copy
	tunnelCopy := *tunnel
	return &tunnelCopy, nil
}

// GetTunnelByName returns a tunnel by VTEP name
func (m *Manager) GetTunnelByName(vtepName string) (*VXLANTunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[vtepName]
	if !exists {
		return nil, fmt.Errorf("tunnel for VTEP %s not found", vtepName)
	}

	// Return a copy
	tunnelCopy := *tunnel
	return &tunnelCopy, nil
}

// ListTunnels returns all configured tunnels
func (m *Manager) ListTunnels() []VXLANTunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels := make([]VXLANTunnel, 0, len(m.tunnels))
	for _, tunnel := range m.tunnels {
		// Return copies
		tunnelCopy := *tunnel
		tunnels = append(tunnels, tunnelCopy)
	}

	return tunnels
}

// GetTunnelStatus returns detailed status of a tunnel
func (m *Manager) GetTunnelStatus(vni int) (VXLANTunnelStatus, error) {
	vtepName := GenerateVTEPName(vni)

	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[vtepName]
	if !exists {
		return VXLANTunnelStatus{}, fmt.Errorf("tunnel for VNI %d not found", vni)
	}

	// Get additional info from kernel if active
	status := VXLANTunnelStatus{
		VNI:        tunnel.Config.VNI,
		VTEPName:   tunnel.Config.VTEPName,
		LocalIP:    tunnel.Config.LocalIP,
		RemoteIP:   tunnel.Config.RemoteIP,
		Status:     tunnel.Status,
		BridgeName: tunnel.BridgeName,
		MTU:        tunnel.Config.MTU,
		CreatedAt:  tunnel.CreatedAt,
		LastError:  tunnel.LastError,
	}

	if tunnel.Status == TunnelStatusActive {
		// Check if interface actually exists
		if _, err := GetVTEP(vtepName); err != nil {
			status.LastError = "VTEP interface not found in kernel"
			status.Status = TunnelStatusFailed
		}
	}

	return status, nil
}

// AddFDBEntry adds an FDB entry to a tunnel
func (m *Manager) AddFDBEntry(vni int, remoteIP string, mac string) error {
	vtepName := GenerateVTEPName(vni)

	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[vtepName]
	if !exists {
		return fmt.Errorf("tunnel for VNI %d not found", vni)
	}

	if tunnel.Status != TunnelStatusActive {
		return fmt.Errorf("tunnel is not active")
	}

	return AddFDBEntry(vtepName, remoteIP, mac)
}

// DeleteFDBEntry removes an FDB entry from a tunnel
func (m *Manager) DeleteFDBEntry(vni int, mac string) error {
	vtepName := GenerateVTEPName(vni)

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.tunnels[vtepName]
	if !exists {
		return fmt.Errorf("tunnel for VNI %d not found", vni)
	}

	return DeleteFDBEntry(vtepName, mac)
}

// ListFDBEntries lists all FDB entries for a tunnel
func (m *Manager) ListFDBEntries(vni int) ([]FDBEntry, error) {
	vtepName := GenerateVTEPName(vni)

	m.mu.RLock()
	defer m.mu.RUnlock()

	t, exists := m.tunnels[vtepName]
	if !exists {
		return nil, fmt.Errorf("tunnel for VNI %d not found", vni)
	}

	if t.Status != TunnelStatusActive {
		return nil, fmt.Errorf("tunnel is not active")
	}

	return ListFDBEntries(vtepName)
}

// RefreshTunnelState refreshes the state of all tunnels by checking kernel state
func (m *Manager) RefreshTunnelState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for vtepName, tunnel := range m.tunnels {
		if tunnel.Status != TunnelStatusActive {
			continue
		}

		// Check if interface still exists
		_, err := GetVTEP(vtepName)
		if err != nil {
			if _, ok := err.(*VTEPNotFoundError); ok {
				tunnel.Status = TunnelStatusFailed
				tunnel.LastError = "VTEP interface removed externally"
				log.Printf("[vxlan] tunnel %s marked as failed: VTEP removed externally", vtepName)
			}
		}
	}
}

// StartHealthCheck starts a background health check for tunnels
func (m *Manager) StartHealthCheck(interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.RefreshTunnelState()
			}
		}
	}()

	log.Printf("[vxlan] health check started with interval %v", interval)
}

// GetBridgeName returns the bridge name for a given VNI
func (m *Manager) GetBridgeName(vni int) (string, error) {
	vtepName := GenerateVTEPName(vni)

	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, exists := m.tunnels[vtepName]
	if !exists {
		return "", fmt.Errorf("tunnel for VNI %d not found", vni)
	}

	return tunnel.BridgeName, nil
}

// IsTunnelActive checks if a tunnel is active
func (m *Manager) IsTunnelActive(vni int) bool {
	status, err := m.GetTunnelStatus(vni)
	if err != nil {
		return false
	}
	return status.Status == TunnelStatusActive
}

// TunnelCount returns the number of managed tunnels
func (m *Manager) TunnelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tunnels)
}
