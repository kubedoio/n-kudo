package mtls

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/enroll"
	"github.com/kubedoio/n-kudo/internal/edge/logger"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

// Default rotation thresholds
const (
	DefaultRotationThreshold  = 0.20 // Rotate when < 20% of lifetime remains
	DefaultMinRotationWindow  = 6 * time.Hour
	DefaultCheckInterval      = 5 * time.Minute
)

// CertificateManager interface for certificate operations
type CertificateManager interface {
	LoadCertificate(paths PKIPaths) (*x509.Certificate, error)
	GenerateCSR(commonName string) (csrPEM []byte, keyPEM []byte, err error)
	WritePKI(paths PKIPaths, keyPEM, certPEM, caPEM []byte) error
}

// RenewClient interface for certificate renewal operations
type RenewClient interface {
	RenewCertificate(ctx context.Context, agentID, csrPEM, refreshToken string) (*enroll.RenewResponse, error)
}

// CertRotator handles automatic certificate rotation before expiry
type CertRotator struct {
	// Configuration
	threshold      float64       // Percentage of lifetime remaining to trigger rotation (e.g., 0.20 for 20%)
	minWindow      time.Duration // Minimum time before expiry to trigger rotation
	checkInterval  time.Duration // How often to check certificate expiry
	
	// Dependencies
	pkiPaths    PKIPaths
	identity    state.Identity
	client      RenewClient
	certManager CertificateManager
	
	// State
	mu           sync.RWMutex
	certExpiry   time.Time
	lastRotation time.Time
	running      bool
	stopCh       chan struct{}
}

// CertRotatorOption configures the CertRotator
type CertRotatorOption func(*CertRotator)

// WithThreshold sets the rotation threshold (percentage of lifetime remaining)
func WithThreshold(threshold float64) CertRotatorOption {
	return func(cr *CertRotator) {
		cr.threshold = threshold
	}
}

// WithMinWindow sets the minimum rotation window
func WithMinWindow(window time.Duration) CertRotatorOption {
	return func(cr *CertRotator) {
		cr.minWindow = window
	}
}

// WithCheckInterval sets the check interval
func WithCheckInterval(interval time.Duration) CertRotatorOption {
	return func(cr *CertRotator) {
		cr.checkInterval = interval
	}
}

// WithCertificateManager sets a custom certificate manager (for testing)
func WithCertificateManager(cm CertificateManager) CertRotatorOption {
	return func(cr *CertRotator) {
		cr.certManager = cm
	}
}

// NewCertRotator creates a new certificate rotator
func NewCertRotator(pkiPaths PKIPaths, identity state.Identity, client RenewClient, opts ...CertRotatorOption) *CertRotator {
	cr := &CertRotator{
		pkiPaths:      pkiPaths,
		identity:      identity,
		client:        client,
		certManager:   &defaultCertManager{},
		threshold:     DefaultRotationThreshold,
		minWindow:     DefaultMinRotationWindow,
		checkInterval: DefaultCheckInterval,
		stopCh:        make(chan struct{}),
	}
	
	for _, opt := range opts {
		opt(cr)
	}
	
	return cr
}

// Start begins the certificate rotation monitoring loop
func (cr *CertRotator) Start(ctx context.Context) error {
	cr.mu.Lock()
	if cr.running {
		cr.mu.Unlock()
		return fmt.Errorf("cert rotator already running")
	}
	cr.running = true
	cr.mu.Unlock()
	
	logger.Info("Certificate rotator started")
	
	// Perform initial check
	if err := cr.checkAndRotate(ctx); err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Initial certificate check failed")
	}
	
	// Start periodic checks
	go cr.run(ctx)
	
	return nil
}

// Stop stops the certificate rotator
func (cr *CertRotator) Stop() {
	cr.mu.Lock()
	if !cr.running {
		cr.mu.Unlock()
		return
	}
	cr.running = false
	cr.mu.Unlock()
	
	close(cr.stopCh)
	logger.Info("Certificate rotator stopped")
}

// run is the main loop for periodic certificate checks
func (cr *CertRotator) run(ctx context.Context) {
	ticker := time.NewTicker(cr.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cr.stopCh:
			return
		case <-ticker.C:
			if err := cr.checkAndRotate(ctx); err != nil {
				logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Warn("Certificate rotation check failed")
			}
		}
	}
}

// checkAndRotate checks certificate expiry and rotates if needed
func (cr *CertRotator) checkAndRotate(ctx context.Context) error {
	cert, err := cr.certManager.LoadCertificate(cr.pkiPaths)
	if err != nil {
		return fmt.Errorf("load certificate: %w", err)
	}
	
	cr.mu.Lock()
	cr.certExpiry = cert.NotAfter
	cr.mu.Unlock()
	
	// Check if rotation is needed
	if !cr.shouldRotate(cert) {
		logger.Debug("Certificate rotation not needed")
		return nil
	}
	
	logger.WithFields(map[string]interface{}{
		"expires_at":    cert.NotAfter.Format(time.RFC3339),
		"time_remaining": time.Until(cert.NotAfter).String(),
	}).Info("Certificate approaching expiry, initiating rotation")
	
	// Perform rotation
	if err := cr.rotate(ctx); err != nil {
		return fmt.Errorf("certificate rotation failed: %w", err)
	}
	
	return nil
}

// shouldRotate determines if certificate rotation is needed based on:
// - < 20% of lifetime remaining, OR
// - < 6 hours until expiry (whichever is longer)
func (cr *CertRotator) shouldRotate(cert *x509.Certificate) bool {
	now := time.Now().UTC()
	totalLifetime := cert.NotAfter.Sub(cert.NotBefore)
	elapsed := now.Sub(cert.NotBefore)
	remaining := cert.NotAfter.Sub(now)
	
	if remaining <= 0 {
		return true // Already expired
	}
	
	// Calculate threshold-based rotation time
	thresholdDuration := time.Duration(float64(totalLifetime) * cr.threshold)
	rotateByThreshold := totalLifetime - thresholdDuration
	
	// Use the longer of: threshold-based or minimum window
	rotateWindow := thresholdDuration
	if cr.minWindow > rotateWindow {
		rotateWindow = cr.minWindow
	}
	
	// Check if we're within the rotation window
	shouldRotate := elapsed >= rotateByThreshold || remaining <= cr.minWindow
	
	logger.WithFields(map[string]interface{}{
		"total_lifetime":     totalLifetime.String(),
		"elapsed":            elapsed.String(),
		"remaining":          remaining.String(),
		"threshold_duration": thresholdDuration.String(),
		"min_window":         cr.minWindow.String(),
		"should_rotate":      shouldRotate,
	}).Debug("Certificate rotation check")
	
	_ = rotateByThreshold // Used in calculations above
	
	return shouldRotate
}

// rotate performs the actual certificate rotation
func (cr *CertRotator) rotate(ctx context.Context) error {
	// 1. Generate new keypair and CSR
	csrPEM, newKeyPEM, err := cr.certManager.GenerateCSR(cr.identity.AgentID)
	if err != nil {
		return fmt.Errorf("generate CSR: %w", err)
	}
	
	// 2. Request new certificate from control plane
	resp, err := cr.client.RenewCertificate(ctx, cr.identity.AgentID, string(csrPEM), cr.identity.RefreshToken)
	if err != nil {
		return fmt.Errorf("renew certificate: %w", err)
	}
	
	// 3. Validate new certificate
	newCert, err := parseCertificatePEM([]byte(resp.ClientCertificatePEM))
	if err != nil {
		return fmt.Errorf("parse new certificate: %w", err)
	}
	
	// 4. Test new certificate with a temporary client
	if err := cr.testNewCertificate(newCert, newKeyPEM, []byte(resp.CACertificatePEM)); err != nil {
		return fmt.Errorf("test new certificate: %w", err)
	}
	
	// 5. Atomically replace certificates
	if err := cr.atomicReplace(newKeyPEM, []byte(resp.ClientCertificatePEM), []byte(resp.CACertificatePEM)); err != nil {
		return fmt.Errorf("atomic replace: %w", err)
	}
	
	// 6. Update refresh token if provided
	if resp.RefreshToken != "" {
		cr.identity.RefreshToken = resp.RefreshToken
	}
	
	cr.mu.Lock()
	cr.lastRotation = time.Now().UTC()
	cr.certExpiry = newCert.NotAfter
	cr.mu.Unlock()
	
	logger.WithFields(map[string]interface{}{
		"new_expires_at": newCert.NotAfter.Format(time.RFC3339),
		"serial":         newCert.SerialNumber.String(),
	}).Info("Certificate rotated successfully")
	
	return nil
}

// testNewCertificate validates that the new certificate works
func (cr *CertRotator) testNewCertificate(cert *x509.Certificate, keyPEM, caPEM []byte) error {
	// Basic validation
	if cert.NotAfter.Before(time.Now().UTC()) {
		return fmt.Errorf("new certificate is already expired")
	}
	
	if cert.NotBefore.After(time.Now().UTC().Add(time.Minute)) {
		return fmt.Errorf("new certificate is not yet valid")
	}
	
	// Parse and validate certificate chain
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return fmt.Errorf("failed to parse CA certificate")
	}
	
	opts := x509.VerifyOptions{
		Roots:       pool,
		CurrentTime: time.Now().UTC(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	
	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}
	
	return nil
}

// atomicReplace atomically replaces certificate files
func (cr *CertRotator) atomicReplace(keyPEM, certPEM, caPEM []byte) error {
	// Ensure PKI directory exists
	if err := os.MkdirAll(cr.pkiPaths.Dir, DirMode); err != nil {
		return fmt.Errorf("create pki dir: %w", err)
	}

	// Create temporary files in the same directory for atomic rename
	tmpKey := cr.pkiPaths.ClientKey + ".tmp"
	tmpCert := cr.pkiPaths.ClientCert + ".tmp"
	tmpCA := cr.pkiPaths.CACert + ".tmp"

	// Clean up temp files on error
	defer func() {
		os.Remove(tmpKey)
		os.Remove(tmpCert)
		os.Remove(tmpCA)
	}()

	// Write new certificates to temp files
	if err := os.WriteFile(tmpKey, keyPEM, PrivateKeyMode); err != nil {
		return fmt.Errorf("write temp key: %w", err)
	}
	if err := os.WriteFile(tmpCert, certPEM, CertMode); err != nil {
		return fmt.Errorf("write temp cert: %w", err)
	}
	if err := os.WriteFile(tmpCA, caPEM, CertMode); err != nil {
		return fmt.Errorf("write temp CA: %w", err)
	}

	// Move files atomically
	if err := os.Rename(tmpKey, cr.pkiPaths.ClientKey); err != nil {
		return fmt.Errorf("replace key file: %w", err)
	}
	if err := os.Rename(tmpCert, cr.pkiPaths.ClientCert); err != nil {
		return fmt.Errorf("replace cert file: %w", err)
	}
	if err := os.Rename(tmpCA, cr.pkiPaths.CACert); err != nil {
		return fmt.Errorf("replace CA file: %w", err)
	}

	return nil
}

// GetStatus returns the current rotation status
func (cr *CertRotator) GetStatus() RotatorStatus {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	return RotatorStatus{
		CertExpiry:   cr.certExpiry,
		LastRotation: cr.lastRotation,
		Running:      cr.running,
	}
}

// RotatorStatus contains the current rotation status
type RotatorStatus struct {
	CertExpiry   time.Time `json:"cert_expiry"`
	LastRotation time.Time `json:"last_rotation"`
	Running      bool      `json:"running"`
}

// defaultCertManager implements CertificateManager using the standard mtls functions
type defaultCertManager struct{}

func (d *defaultCertManager) LoadCertificate(paths PKIPaths) (*x509.Certificate, error) {
	return LoadCertificate(paths.ClientCert)
}

func (d *defaultCertManager) GenerateCSR(commonName string) (csrPEM []byte, keyPEM []byte, err error) {
	key, err := GeneratePrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}
	
	csrPEM, err = GenerateCSRPEM(key, commonName)
	if err != nil {
		return nil, nil, fmt.Errorf("generate CSR: %w", err)
	}
	
	return csrPEM, EncodePrivateKeyPEM(key), nil
}

func (d *defaultCertManager) WritePKI(paths PKIPaths, keyPEM, certPEM, caPEM []byte) error {
	return WritePKI(paths, keyPEM, certPEM, caPEM)
}

// LoadCertificate loads and parses a certificate from file
func LoadCertificate(certPath string) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}
	
	return parseCertificatePEM(certPEM)
}

func parseCertificatePEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	
	return cert, nil
}

// GetCertificateExpiry returns the expiry time of the certificate at the given path
func GetCertificateExpiry(certPath string) (time.Time, error) {
	cert, err := LoadCertificate(certPath)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}
