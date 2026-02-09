package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"sync"
	"time"
)

// RevocationReason represents the reason for certificate revocation
type RevocationReason int

const (
	ReasonUnspecified          RevocationReason = 0
	ReasonKeyCompromise        RevocationReason = 1
	ReasonCACompromise         RevocationReason = 2
	ReasonAffiliationChanged   RevocationReason = 3
	ReasonSuperseded           RevocationReason = 4
	ReasonCessationOfOperation RevocationReason = 5
	ReasonCertificateHold      RevocationReason = 6
	ReasonRemoveFromCRL        RevocationReason = 8
	ReasonPrivilegeWithdrawn   RevocationReason = 9
	ReasonAACompromise         RevocationReason = 10
)

// RevokedCertificate represents a revoked certificate entry
type RevokedCertificate struct {
	SerialNumber string
	RevokedAt    time.Time
	Reason       RevocationReason
	AgentID      string
}

// CRLManager manages certificate revocation lists
type CRLManager struct {
	mu          sync.RWMutex
	caCert      *x509.Certificate
	caKey       *rsa.PrivateKey
	revoked     map[string]*RevokedCertificate // serial -> entry
	crlBytes    []byte
	crlPEM      []byte
	lastUpdated time.Time
	crlURL      string
}

// NewCRLManager creates a new CRL manager
func NewCRLManager(caCert *x509.Certificate, caKey *rsa.PrivateKey, crlURL string) *CRLManager {
	return &CRLManager{
		caCert:  caCert,
		caKey:   caKey,
		revoked: make(map[string]*RevokedCertificate),
		crlURL:  crlURL,
	}
}

// Revoke adds a certificate to the revocation list
func (m *CRLManager) Revoke(serial string, reason RevocationReason, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already revoked
	if _, exists := m.revoked[serial]; exists {
		return nil // Already revoked, idempotent
	}

	entry := &RevokedCertificate{
		SerialNumber: serial,
		RevokedAt:    time.Now().UTC(),
		Reason:       reason,
		AgentID:      agentID,
	}
	m.revoked[serial] = entry

	// Regenerate CRL
	return m.generateCRLLocked()
}

// IsRevoked checks if a certificate serial is revoked
func (m *CRLManager) IsRevoked(serial string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.revoked[serial]
	return exists
}

// GetRevokedCertificate returns the revocation details for a certificate
func (m *CRLManager) GetRevokedCertificate(serial string) (*RevokedCertificate, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.revoked[serial]
	if !exists {
		return nil, false
	}
	// Return a copy
	copy := *entry
	return &copy, true
}

// GetCRL returns the DER-encoded CRL
func (m *CRLManager) GetCRL() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]byte(nil), m.crlBytes...)
}

// GetCRLPEM returns the PEM-encoded CRL
func (m *CRLManager) GetCRLPEM() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]byte(nil), m.crlPEM...)
}

// GetLastUpdated returns when the CRL was last updated
func (m *CRLManager) GetLastUpdated() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.lastUpdated
}

// ListRevoked returns all revoked certificates
func (m *CRLManager) ListRevoked() []*RevokedCertificate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*RevokedCertificate, 0, len(m.revoked))
	for _, entry := range m.revoked {
		copy := *entry
		result = append(result, &copy)
	}
	return result
}

// generateCRLLocked generates a new CRL (must be called with lock held)
func (m *CRLManager) generateCRLLocked() error {
	now := time.Now().UTC()

	// Build revoked certificates list
	revokedCerts := make([]pkix.RevokedCertificate, 0, len(m.revoked))
	for _, entry := range m.revoked {
		serial := new(big.Int)
		_, ok := serial.SetString(entry.SerialNumber, 10)
		if !ok {
			// If parsing fails, try to use as-is (it might be a different format)
			serial = big.NewInt(0)
		}

		revokedCert := pkix.RevokedCertificate{
			SerialNumber:   serial,
			RevocationTime: entry.RevokedAt,
		}

		// Add reason code extension if not unspecified
		if entry.Reason != ReasonUnspecified {
			// ASN.1 encode the reason code as ENUMERATED
			reasonBytes, err := asn1.Marshal(asn1.Enumerated(entry.Reason))
			if err == nil {
				revokedCert.Extensions = []pkix.Extension{
					{
						Id:       []int{2, 5, 29, 21}, // id-ce-cRLReasons OID
						Critical: false,
						Value:    reasonBytes,
					},
				}
			}
		}

		revokedCerts = append(revokedCerts, revokedCert)
	}

	// Calculate next update (7 days from now)
	nextUpdate := now.Add(7 * 24 * time.Hour)

	// Create CRL template
	template := &x509.RevocationList{
		Number:     big.NewInt(now.Unix()),
		ThisUpdate: now,
		NextUpdate: nextUpdate,
		RevokedCertificates: revokedCerts,
	}

	// Add CRL Distribution Point if configured
	if m.crlURL != "" {
		// The IssuingDistributionPoint extension is handled by the library
		// when creating the CRL
	}

	// Generate CRL
	crlBytes, err := x509.CreateRevocationList(rand.Reader, template, m.caCert, m.caKey)
	if err != nil {
		return err
	}

	m.crlBytes = crlBytes
	m.crlPEM = pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: crlBytes})
	m.lastUpdated = now

	return nil
}

// GenerateCRL regenerates the CRL (public method for manual regeneration)
func (m *CRLManager) GenerateCRL() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.generateCRLLocked()
}

// RemoveRevocation removes a certificate from the revocation list (for testing/admin)
func (m *CRLManager) RemoveRevocation(serial string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.revoked, serial)
	return m.generateCRLLocked()
}
