package mtls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/enroll"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

// mockCertManager is a mock implementation of CertificateManager for testing
type mockCertManager struct {
	cert         *x509.Certificate
	loadErr      error
	generateErr  error
	writeErr     error
	writtenPKI   *PKIPaths
	writtenKey   []byte
	writtenCert  []byte
	writtenCA    []byte
}

func (m *mockCertManager) LoadCertificate(paths PKIPaths) (*x509.Certificate, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.cert, nil
}

func (m *mockCertManager) GenerateCSR(commonName string) (csrPEM []byte, keyPEM []byte, err error) {
	if m.generateErr != nil {
		return nil, nil, m.generateErr
	}
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	csr := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: commonName},
	}
	csrDER, _ := x509.CreateCertificateRequest(rand.Reader, csr, key)
	csrPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return csrPEM, keyPEM, nil
}

func (m *mockCertManager) WritePKI(paths PKIPaths, keyPEM, certPEM, caPEM []byte) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writtenPKI = &paths
	m.writtenKey = keyPEM
	m.writtenCert = certPEM
	m.writtenCA = caPEM
	return nil
}

// mockRenewClient is a mock implementation of RenewClient for testing
type mockRenewClient struct {
	resp    *enroll.RenewResponse
	err     error
	called  bool
	agentID string
	csrPEM  string
	token   string
}

func (m *mockRenewClient) RenewCertificate(ctx context.Context, agentID, csrPEM, refreshToken string) (*enroll.RenewResponse, error) {
	m.called = true
	m.agentID = agentID
	m.csrPEM = csrPEM
	m.token = refreshToken
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func generateTestCert(notBefore, notAfter time.Time) *x509.Certificate {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject:      pkix.Name{CommonName: "test-agent"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	cert, _ := x509.ParseCertificate(certDER)
	return cert
}

func TestShouldRotate(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		notBefore time.Time
		notAfter  time.Time
		want      bool
	}{
		{
			name:      "fresh certificate (80% lifetime remaining)",
			notBefore: now.Add(-24 * time.Hour),
			notAfter:  now.Add(96 * time.Hour), // 5 day total lifetime, 4 days remaining (80%)
			want:      false,
		},
		{
			name:      "approaching threshold (15% lifetime remaining)",
			notBefore: now.Add(-85 * time.Hour),
			notAfter:  now.Add(15 * time.Hour), // 100 hour total lifetime, 15% remaining
			want:      true,
		},
		{
			name:      "within min window (5 hours left)",
			notBefore: now.Add(-19 * time.Hour),
			notAfter:  now.Add(5 * time.Hour), // 24 hour total lifetime, 25% remaining but < 6 hours
			want:      true,
		},
		{
			name:      "exactly at threshold (20% lifetime remaining)",
			notBefore: now.Add(-80 * time.Hour),
			notAfter:  now.Add(20 * time.Hour), // 100 hour total lifetime, 20% remaining
			want:      true, // 80% elapsed, should rotate at threshold
		},
		{
			name:      "already expired",
			notBefore: now.Add(-48 * time.Hour),
			notAfter:  now.Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "short lived cert within min window",
			notBefore: now.Add(-2 * time.Hour),
			notAfter:  now.Add(4 * time.Hour), // 6 hour total lifetime
			want:      true, // less than 6 hours remaining
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := generateTestCert(tt.notBefore, tt.notAfter)
			cr := NewCertRotator(PKIPaths{}, state.Identity{}, nil)

			got := cr.shouldRotate(cert)
			if got != tt.want {
				t.Errorf("shouldRotate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAndRotate_NoRotationNeeded(t *testing.T) {
	now := time.Now().UTC()
	// Fresh certificate, no rotation needed
	mockCert := generateTestCert(now.Add(-24*time.Hour), now.Add(96*time.Hour))
	
	mockCM := &mockCertManager{cert: mockCert}
	mockClient := &mockRenewClient{}
	
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{AgentID: "test-agent"},
		mockClient,
		WithCertificateManager(mockCM),
	)
	
	ctx := context.Background()
	err := cr.checkAndRotate(ctx)
	
	if err != nil {
		t.Errorf("checkAndRotate() error = %v", err)
	}
	
	if mockClient.called {
		t.Error("RenewCertificate should not have been called")
	}
}

func TestCheckAndRotate_RotationNeeded(t *testing.T) {
	now := time.Now().UTC()
	// Certificate within rotation window
	mockCert := generateTestCert(now.Add(-85*time.Hour), now.Add(15*time.Hour))

	// Generate a proper CA certificate that can sign other certificates
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caCertDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caCertDER)
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	// Generate a new client certificate signed by the CA
	clientKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-agent"},
		NotBefore:    now,
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientCertDER, _ := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})

	mockCM := &mockCertManager{cert: mockCert}
	mockClient := &mockRenewClient{
		resp: &enroll.RenewResponse{
			ClientCertificatePEM: string(clientCertPEM),
			CACertificatePEM:     string(caCertPEM),
			RefreshToken:         "new-refresh-token",
		},
	}

	tmpDir := t.TempDir()
	pkiPaths := PKIPaths{
		Dir:        tmpDir,
		ClientKey:  filepath.Join(tmpDir, "client.key"),
		ClientCert: filepath.Join(tmpDir, "client.crt"),
		CACert:     filepath.Join(tmpDir, "ca.crt"),
	}

	// Create initial cert files
	os.WriteFile(pkiPaths.ClientKey, []byte("key"), 0600)
	os.WriteFile(pkiPaths.ClientCert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: mockCert.Raw}), 0644)
	os.WriteFile(pkiPaths.CACert, caCertPEM, 0644)

	cr := NewCertRotator(
		pkiPaths,
		state.Identity{AgentID: "test-agent", RefreshToken: "old-token"},
		mockClient,
		WithCertificateManager(mockCM),
	)

	ctx := context.Background()
	err := cr.checkAndRotate(ctx)

	if err != nil {
		t.Errorf("checkAndRotate() error = %v", err)
	}

	if !mockClient.called {
		t.Error("RenewCertificate should have been called")
	}

	if mockClient.agentID != "test-agent" {
		t.Errorf("agentID = %v, want test-agent", mockClient.agentID)
	}

	if mockClient.token != "old-token" {
		t.Errorf("token = %v, want old-token", mockClient.token)
	}
}

func TestCheckAndRotate_LoadError(t *testing.T) {
	mockCM := &mockCertManager{loadErr: errors.New("file not found")}
	mockClient := &mockRenewClient{}
	
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{},
		mockClient,
		WithCertificateManager(mockCM),
	)
	
	ctx := context.Background()
	err := cr.checkAndRotate(ctx)
	
	if err == nil {
		t.Error("Expected error for load failure")
	}
	
	if mockClient.called {
		t.Error("RenewCertificate should not have been called on load error")
	}
}

func TestCheckAndRotate_RenewError(t *testing.T) {
	now := time.Now().UTC()
	mockCert := generateTestCert(now.Add(-85*time.Hour), now.Add(15*time.Hour))
	
	mockCM := &mockCertManager{cert: mockCert}
	mockClient := &mockRenewClient{err: errors.New("renewal failed")}
	
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{AgentID: "test-agent"},
		mockClient,
		WithCertificateManager(mockCM),
	)
	
	ctx := context.Background()
	err := cr.checkAndRotate(ctx)
	
	if err == nil {
		t.Error("Expected error for renewal failure")
	}
}

func TestRotate_InvalidCertificateResponse(t *testing.T) {
	now := time.Now().UTC()
	mockCert := generateTestCert(now.Add(-85*time.Hour), now.Add(15*time.Hour))
	
	mockCM := &mockCertManager{cert: mockCert}
	mockClient := &mockRenewClient{
		resp: &enroll.RenewResponse{
			ClientCertificatePEM: "invalid-pem",
			CACertificatePEM:     "invalid-ca",
		},
	}
	
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{AgentID: "test-agent"},
		mockClient,
		WithCertificateManager(mockCM),
	)
	
	ctx := context.Background()
	err := cr.checkAndRotate(ctx)
	
	if err == nil {
		t.Error("Expected error for invalid certificate response")
	}
}

func TestCertRotator_StartStop(t *testing.T) {
	now := time.Now().UTC()
	mockCert := generateTestCert(now.Add(-24*time.Hour), now.Add(96*time.Hour))
	mockCM := &mockCertManager{cert: mockCert}
	mockClient := &mockRenewClient{}
	
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{},
		mockClient,
		WithCertificateManager(mockCM),
		WithCheckInterval(100*time.Millisecond),
	)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start the rotator
	err := cr.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}
	
	// Verify it's running
	status := cr.GetStatus()
	if !status.Running {
		t.Error("Expected rotator to be running")
	}
	
	// Try to start again (should fail)
	err = cr.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running rotator")
	}
	
	// Stop the rotator
	cr.Stop()
	
	// Verify it's stopped
	status = cr.GetStatus()
	if status.Running {
		t.Error("Expected rotator to be stopped")
	}
}

func TestTestNewCertificate(t *testing.T) {
	now := time.Now().UTC()

	// Create a proper CA certificate
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caCertDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caCertDER)
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	// Create a client key for testing
	clientKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})

	tests := []struct {
		name      string
		notBefore time.Time
		notAfter  time.Time
		wantErr   bool
	}{
		{
			name:      "valid certificate",
			notBefore: now.Add(-1 * time.Hour),
			notAfter:  now.Add(24 * time.Hour),
			wantErr:   false,
		},
		{
			name:      "already expired",
			notBefore: now.Add(-48 * time.Hour),
			notAfter:  now.Add(-1 * time.Hour),
			wantErr:   true,
		},
		{
			name:      "not yet valid",
			notBefore: now.Add(2 * time.Hour),
			notAfter:  now.Add(24 * time.Hour),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a client certificate signed by the CA
			clientTemplate := &x509.Certificate{
				SerialNumber: big.NewInt(2),
				Subject:      pkix.Name{CommonName: "test-agent"},
				NotBefore:    tt.notBefore,
				NotAfter:     tt.notAfter,
				KeyUsage:     x509.KeyUsageDigitalSignature,
				ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}
			clientCertDER, _ := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
			cert, _ := x509.ParseCertificate(clientCertDER)

			cr := NewCertRotator(PKIPaths{}, state.Identity{}, nil)
			err := cr.testNewCertificate(cert, clientKeyPEM, caCertPEM)

			if (err != nil) != tt.wantErr {
				t.Errorf("testNewCertificate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAtomicReplace(t *testing.T) {
	tmpDir := t.TempDir()
	pkiPaths := PKIPaths{
		Dir:        tmpDir,
		ClientKey:  filepath.Join(tmpDir, "client.key"),
		ClientCert: filepath.Join(tmpDir, "client.crt"),
		CACert:     filepath.Join(tmpDir, "ca.crt"),
	}

	// Create initial files
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(pkiPaths.ClientKey, []byte("old-key"), 0600)
	os.WriteFile(pkiPaths.ClientCert, []byte("old-cert"), 0644)
	os.WriteFile(pkiPaths.CACert, []byte("old-ca"), 0644)

	mockCM := &mockCertManager{}
	cr := NewCertRotator(
		pkiPaths,
		state.Identity{},
		nil,
		WithCertificateManager(mockCM),
	)

	newKey := []byte("new-key")
	newCert := []byte("new-cert")
	newCA := []byte("new-ca")

	err := cr.atomicReplace(newKey, newCert, newCA)
	if err != nil {
		t.Errorf("atomicReplace() error = %v", err)
	}

	// Verify files were replaced
	keyContent, _ := os.ReadFile(pkiPaths.ClientKey)
	if string(keyContent) != "new-key" {
		t.Errorf("Key file not replaced, got: %s", string(keyContent))
	}

	certContent, _ := os.ReadFile(pkiPaths.ClientCert)
	if string(certContent) != "new-cert" {
		t.Errorf("Cert file not replaced, got: %s", string(certContent))
	}

	caContent, _ := os.ReadFile(pkiPaths.CACert)
	if string(caContent) != "new-ca" {
		t.Errorf("CA file not replaced, got: %s", string(caContent))
	}

	// Verify temp files are cleaned up
	if _, err := os.Stat(pkiPaths.ClientKey + ".tmp"); !os.IsNotExist(err) {
		t.Error("Temp key file should be cleaned up")
	}
}

func TestLoadCertificate(t *testing.T) {
	now := time.Now().UTC()
	cert := generateTestCert(now.Add(-1*time.Hour), now.Add(24*time.Hour))
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.crt")
	
	// Test file not found
	_, err := LoadCertificate(certPath)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	
	// Write valid certificate
	os.WriteFile(certPath, certPEM, 0644)
	
	loadedCert, err := LoadCertificate(certPath)
	if err != nil {
		t.Errorf("LoadCertificate() error = %v", err)
	}
	
	if loadedCert.SerialNumber.Cmp(cert.SerialNumber) != 0 {
		t.Error("Loaded certificate serial number mismatch")
	}
	
	// Test invalid PEM
	os.WriteFile(certPath, []byte("not valid pem"), 0644)
	_, err = LoadCertificate(certPath)
	if err == nil {
		t.Error("Expected error for invalid PEM")
	}
}

func TestGetCertificateExpiry(t *testing.T) {
	now := time.Now().UTC()
	expiry := now.Add(24 * time.Hour)
	cert := generateTestCert(now.Add(-1*time.Hour), expiry)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.crt")
	os.WriteFile(certPath, certPEM, 0644)
	
	gotExpiry, err := GetCertificateExpiry(certPath)
	if err != nil {
		t.Errorf("GetCertificateExpiry() error = %v", err)
	}
	
	// Allow small time difference due to parsing
	if gotExpiry.Sub(expiry) > time.Second || expiry.Sub(gotExpiry) > time.Second {
		t.Errorf("GetCertificateExpiry() = %v, want %v", gotExpiry, expiry)
	}
}

func TestGetCertificateExpiry_NotFound(t *testing.T) {
	_, err := GetCertificateExpiry("/nonexistent/path/cert.crt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCertRotatorOptions(t *testing.T) {
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{},
		nil,
		WithThreshold(0.30),
		WithMinWindow(12*time.Hour),
		WithCheckInterval(10*time.Minute),
	)
	
	if cr.threshold != 0.30 {
		t.Errorf("threshold = %v, want 0.30", cr.threshold)
	}
	
	if cr.minWindow != 12*time.Hour {
		t.Errorf("minWindow = %v, want 12h", cr.minWindow)
	}
	
	if cr.checkInterval != 10*time.Minute {
		t.Errorf("checkInterval = %v, want 10m", cr.checkInterval)
	}
}

func TestCertRotatorStatus(t *testing.T) {
	now := time.Now().UTC()
	mockCert := generateTestCert(now.Add(-24*time.Hour), now.Add(96*time.Hour))
	mockCM := &mockCertManager{cert: mockCert}
	
	cr := NewCertRotator(
		PKIPaths{},
		state.Identity{},
		nil,
		WithCertificateManager(mockCM),
	)
	
	status := cr.GetStatus()
	if status.Running {
		t.Error("Expected rotator to not be running initially")
	}
	
	// After checkAndRotate, expiry should be set
	ctx := context.Background()
	cr.checkAndRotate(ctx)
	
	status = cr.GetStatus()
	if status.CertExpiry.IsZero() {
		t.Error("Expected CertExpiry to be set after check")
	}
}
