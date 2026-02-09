package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func createTestCA(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("failed to generate serial: %v", err)
	}

	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test"},
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, key
}

func TestCRLManager_Revoke(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	serial := "1234567890"
	agentID := "agent-1"

	// Revoke a certificate
	err := manager.Revoke(serial, ReasonKeyCompromise, agentID)
	if err != nil {
		t.Fatalf("failed to revoke certificate: %v", err)
	}

	// Check if revoked
	if !manager.IsRevoked(serial) {
		t.Error("expected certificate to be revoked")
	}

	// Check revocation details
	entry, exists := manager.GetRevokedCertificate(serial)
	if !exists {
		t.Fatal("expected revocation entry to exist")
	}
	if entry.SerialNumber != serial {
		t.Errorf("expected serial %s, got %s", serial, entry.SerialNumber)
	}
	if entry.AgentID != agentID {
		t.Errorf("expected agent ID %s, got %s", agentID, entry.AgentID)
	}
	if entry.Reason != ReasonKeyCompromise {
		t.Errorf("expected reason %d, got %d", ReasonKeyCompromise, entry.Reason)
	}
}

func TestCRLManager_IsRevoked_NotRevoked(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	// Check non-revoked certificate
	if manager.IsRevoked("not-revoked") {
		t.Error("expected certificate to not be revoked")
	}
}

func TestCRLManager_DoubleRevoke(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	serial := "1234567890"

	// Revoke same certificate twice (should be idempotent)
	err := manager.Revoke(serial, ReasonKeyCompromise, "agent-1")
	if err != nil {
		t.Fatalf("failed to revoke certificate: %v", err)
	}

	err = manager.Revoke(serial, ReasonSuperseded, "agent-2")
	if err != nil {
		t.Fatalf("failed to revoke certificate again: %v", err)
	}

	// Should still be revoked with original details
	entry, exists := manager.GetRevokedCertificate(serial)
	if !exists {
		t.Fatal("expected revocation entry to exist")
	}
	if entry.Reason != ReasonKeyCompromise {
		t.Error("expected original revocation reason to be preserved")
	}
}

func TestCRLManager_GetCRL(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	// Revoke some certificates
	serials := []string{"1", "2", "3"}
	for i, serial := range serials {
		err := manager.Revoke(serial, RevocationReason(i+1), "agent-"+serial)
		if err != nil {
			t.Fatalf("failed to revoke certificate %s: %v", serial, err)
		}
	}

	// Get CRL
	crlBytes := manager.GetCRL()
	if len(crlBytes) == 0 {
		t.Fatal("expected CRL bytes")
	}

	// Parse and verify CRL
	crl, err := x509.ParseRevocationList(crlBytes)
	if err != nil {
		t.Fatalf("failed to parse CRL: %v", err)
	}

	if len(crl.RevokedCertificates) != 3 {
		t.Errorf("expected 3 revoked certificates, got %d", len(crl.RevokedCertificates))
	}
}

func TestCRLManager_GetCRLPEM(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	// Revoke a certificate
	err := manager.Revoke("123", ReasonKeyCompromise, "agent-1")
	if err != nil {
		t.Fatalf("failed to revoke certificate: %v", err)
	}

	// Get PEM CRL
	crlPEM := manager.GetCRLPEM()
	if len(crlPEM) == 0 {
		t.Fatal("expected CRL PEM")
	}

	// Verify it starts with PEM header
	if string(crlPEM[:11]) != "-----BEGIN " {
		t.Error("expected PEM to start with -----BEGIN ")
	}
}

func TestCRLManager_ListRevoked(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	// Initially empty
	if len(manager.ListRevoked()) != 0 {
		t.Error("expected empty revoked list initially")
	}

	// Revoke some certificates
	serials := []string{"1", "2", "3"}
	for _, serial := range serials {
		err := manager.Revoke(serial, ReasonKeyCompromise, "agent-"+serial)
		if err != nil {
			t.Fatalf("failed to revoke certificate %s: %v", serial, err)
		}
	}

	// List revoked
	revoked := manager.ListRevoked()
	if len(revoked) != 3 {
		t.Errorf("expected 3 revoked certificates, got %d", len(revoked))
	}
}

func TestCRLManager_RemoveRevocation(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	serial := "1234567890"

	// Revoke and then remove
	err := manager.Revoke(serial, ReasonKeyCompromise, "agent-1")
	if err != nil {
		t.Fatalf("failed to revoke certificate: %v", err)
	}

	if !manager.IsRevoked(serial) {
		t.Error("expected certificate to be revoked")
	}

	err = manager.RemoveRevocation(serial)
	if err != nil {
		t.Fatalf("failed to remove revocation: %v", err)
	}

	if manager.IsRevoked(serial) {
		t.Error("expected certificate to not be revoked after removal")
	}
}

func TestCRLManager_GetLastUpdated(t *testing.T) {
	caCert, caKey := createTestCA(t)
	manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

	// Initially zero
	if !manager.GetLastUpdated().IsZero() {
		t.Error("expected last updated to be zero initially")
	}

	// Revoke a certificate
	before := time.Now().UTC()
	err := manager.Revoke("123", ReasonKeyCompromise, "agent-1")
	if err != nil {
		t.Fatalf("failed to revoke certificate: %v", err)
	}
	after := time.Now().UTC()

	lastUpdated := manager.GetLastUpdated()
	if lastUpdated.Before(before) || lastUpdated.After(after) {
		t.Error("expected last updated to be between before and after")
	}
}

func TestRevocationReasons(t *testing.T) {
	tests := []struct {
		reason RevocationReason
		name   string
	}{
		{ReasonUnspecified, "Unspecified"},
		{ReasonKeyCompromise, "KeyCompromise"},
		{ReasonCACompromise, "CACompromise"},
		{ReasonAffiliationChanged, "AffiliationChanged"},
		{ReasonSuperseded, "Superseded"},
		{ReasonCessationOfOperation, "CessationOfOperation"},
		{ReasonCertificateHold, "CertificateHold"},
		{ReasonRemoveFromCRL, "RemoveFromCRL"},
		{ReasonPrivilegeWithdrawn, "PrivilegeWithdrawn"},
		{ReasonAACompromise, "AACompromise"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caCert, caKey := createTestCA(t)
			manager := NewCRLManager(caCert, caKey, "http://example.com/crl")

			err := manager.Revoke("123", tt.reason, "agent-1")
			if err != nil {
				t.Fatalf("failed to revoke with reason %s: %v", tt.name, err)
			}

			entry, exists := manager.GetRevokedCertificate("123")
			if !exists {
				t.Fatal("expected revocation entry to exist")
			}
			if entry.Reason != tt.reason {
				t.Errorf("expected reason %d, got %d", tt.reason, entry.Reason)
			}
		})
	}
}
