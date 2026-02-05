package mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	DirMode        = 0o700
	PrivateKeyMode = 0o600
	CertMode       = 0o644
)

type PKIPaths struct {
	Dir        string
	ClientKey  string
	ClientCert string
	CACert     string
}

func DefaultPKIPaths(dir string) PKIPaths {
	return PKIPaths{
		Dir:        dir,
		ClientKey:  filepath.Join(dir, "client.key"),
		ClientCert: filepath.Join(dir, "client.crt"),
		CACert:     filepath.Join(dir, "ca.crt"),
	}
}

func GeneratePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 4096)
}

func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

func GenerateCSRPEM(key *rsa.PrivateKey, commonName string) ([]byte, error) {
	tpl := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: commonName,
		},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}), nil
}

func WritePKI(paths PKIPaths, keyPEM, certPEM, caPEM []byte) error {
	if err := os.MkdirAll(paths.Dir, DirMode); err != nil {
		return fmt.Errorf("mkdir pki dir: %w", err)
	}
	if err := os.WriteFile(paths.ClientKey, keyPEM, PrivateKeyMode); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	if err := os.WriteFile(paths.ClientCert, certPEM, CertMode); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(paths.CACert, caPEM, CertMode); err != nil {
		return fmt.Errorf("write ca cert: %w", err)
	}
	if err := os.Chmod(paths.ClientKey, PrivateKeyMode); err != nil {
		return fmt.Errorf("chmod key: %w", err)
	}
	return nil
}

func NewMutualTLSClient(paths PKIPaths, insecureSkipVerify bool) (*http.Client, error) {
	cert, err := tls.LoadX509KeyPair(paths.ClientCert, paths.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("load client keypair: %w", err)
	}

	caPEM, err := os.ReadFile(paths.CACert)
	if err != nil {
		return nil, fmt.Errorf("read ca cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse ca cert")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS13,
			Certificates:       []tls.Certificate{cert},
			RootCAs:            pool,
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return &http.Client{Transport: tr, Timeout: 20 * time.Second}, nil
}

func NewBootstrapTLSClient(caPEM []byte, insecureSkipVerify bool) (*http.Client, error) {
	pool := x509.NewCertPool()
	if len(caPEM) > 0 {
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("parse bootstrap ca cert")
		}
	}

	cfg := &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: insecureSkipVerify}
	if len(caPEM) > 0 {
		cfg.RootCAs = pool
	}

	tr := &http.Transport{TLSClientConfig: cfg}
	return &http.Client{Transport: tr, Timeout: 20 * time.Second}, nil
}

func SelfSignedCA(commonName string) (certPEM []byte, keyPEM []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(now.UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), EncodePrivateKeyPEM(key), nil
}
