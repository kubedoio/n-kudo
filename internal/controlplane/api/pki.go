package controlplane

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"time"
)

type InternalCA struct {
	cert    *x509.Certificate
	key     *rsa.PrivateKey
	certPEM []byte
}

func LoadOrCreateInternalCA(commonName string, requirePersistent bool) (*InternalCA, error) {
	certFile := os.Getenv("CA_CERT_FILE")
	keyFile := os.Getenv("CA_KEY_FILE")
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			return nil, errors.New("CA_CERT_FILE and CA_KEY_FILE must both be set")
		}
		certPEM, err := os.ReadFile(certFile)
		if err != nil {
			return nil, err
		}
		keyPEM, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		cert, key, err := parsePEMCA(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		return &InternalCA{cert: cert, key: key, certPEM: certPEM}, nil
	}
	if requirePersistent {
		return nil, errors.New("REQUIRE_PERSISTENT_PKI=true requires CA_CERT_FILE and CA_KEY_FILE")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"n-kudo"},
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return &InternalCA{cert: cert, key: key, certPEM: certPEM}, nil
}

func (c *InternalCA) CertPEM() []byte {
	return append([]byte(nil), c.certPEM...)
}

func (c *InternalCA) Certificate() *x509.Certificate {
	return c.cert
}

func (c *InternalCA) Key() *rsa.PrivateKey {
	return c.key
}

// CertPool returns a new cert pool containing the CA certificate
func (c *InternalCA) CertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(c.certPEM); !ok {
		return nil
	}
	return pool
}

func (c *InternalCA) SignAgentCSR(csrPEM []byte, agentID, tenantID, siteID string, ttl time.Duration) (certPEM []byte, serial string, err error) {
	_ = tenantID
	_ = siteID
	block, _ := pem.Decode(csrPEM)
	if block == nil {
		return nil, "", errors.New("invalid csr pem")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, "", err
	}
	serialNum, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()

	// Get CRL URL from environment
	crlURL := os.Getenv("CRL_URL")

	tmpl := &x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			CommonName:   agentID,
			Organization: []string{"n-kudo-agent"},
		},
		NotBefore:   now.Add(-1 * time.Minute),
		NotAfter:    now.Add(ttl),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Add CRL Distribution Point if configured
	if crlURL != "" {
		tmpl.CRLDistributionPoints = []string{crlURL}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, csr.PublicKey, c.key)
	if err != nil {
		return nil, "", err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), serialNum.String(), nil
}

func GenerateServerTLSCert(requirePersistent bool) (tls.Certificate, error) {
	certFile := os.Getenv("SERVER_CERT_FILE")
	keyFile := os.Getenv("SERVER_KEY_FILE")
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			return tls.Certificate{}, errors.New("SERVER_CERT_FILE and SERVER_KEY_FILE must both be set")
		}
		return tls.LoadX509KeyPair(certFile, keyFile)
	}
	if requirePersistent {
		return tls.Certificate{}, errors.New("REQUIRE_PERSISTENT_PKI=true requires SERVER_CERT_FILE and SERVER_KEY_FILE")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func parsePEMCA(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, errors.New("invalid ca cert")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("invalid ca key")
	}
	var key *rsa.PrivateKey
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		parsed, perr := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if perr != nil {
			return nil, nil, perr
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, errors.New("unsupported key type")
		}
		key = rsaKey
	default:
		return nil, nil, errors.New("unsupported private key format")
	}
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}
