package integration_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/n-kudo/n-kudo-edge/pkg/controlplane"
	"github.com/n-kudo/n-kudo-edge/pkg/enroll"
	"github.com/n-kudo/n-kudo-edge/pkg/hostfacts"
	"github.com/n-kudo/n-kudo-edge/pkg/logstream"
	"github.com/n-kudo/n-kudo-edge/pkg/mtls"
	"github.com/n-kudo/n-kudo-edge/pkg/netbird"
)

func TestEnrollThenMutualTLSHeartbeatAndLogs(t *testing.T) {
	caCertPEM, _, caCert, caKey := newTestCA(t)
	serverTLSCert := newServerCert(t, caCert, caKey)

	var heartbeatCalls atomic.Int32
	var logCalls atomic.Int32

	ingestMux := http.NewServeMux()
	ingestMux.HandleFunc("/v1/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		heartbeatCalls.Add(1)
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client cert required", http.StatusUnauthorized)
			return
		}
		payload := map[string]any{
			"next_heartbeat_seconds": 15,
			"pending_plans": []any{},
		}
		_ = json.NewEncoder(w).Encode(payload)
	})
	ingestMux.HandleFunc("/v1/plans/next", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"plans": []any{}})
	})
	ingestMux.HandleFunc("/v1/executions/result", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	ingestMux.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		logCalls.Add(1)
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client cert required", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	ingestSrv := httptest.NewUnstartedServer(ingestMux)
	clientCAPool := x509.NewCertPool()
	clientCAPool.AppendCertsFromPEM(caCertPEM)
	ingestSrv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAPool,
		MinVersion:   tls.VersionTLS13,
	}
	ingestSrv.StartTLS()
	defer ingestSrv.Close()

	bootstrapMux := http.NewServeMux()
	bootstrapMux.HandleFunc("/v1/enroll", func(w http.ResponseWriter, r *http.Request) {
		var req enroll.EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		certPEM := signCSR(t, req.CSRPEM, caCert, caKey)
		resp := enroll.EnrollResponse{
			TenantID:             "tenant-1",
			SiteID:               "site-1",
			HostID:               "host-1",
			AgentID:              "agent-1",
			ClientCertificatePEM: certPEM,
			CACertificatePEM:     string(caCertPEM),
			RefreshToken:         "refresh-token",
			HeartbeatEndpoint:    ingestSrv.URL,
			HeartbeatIntervalSec: 15,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	bootstrapSrv := httptest.NewTLSServer(bootstrapMux)
	defer bootstrapSrv.Close()

	bootstrapHTTP, err := mtls.NewBootstrapTLSClient(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	enrollClient := enroll.Client{BaseURL: bootstrapSrv.URL, HTTP: bootstrapHTTP}

	key, err := mtls.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	csr, err := mtls.GenerateCSRPEM(key, "nkudo-edge-test")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := enrollClient.Enroll(context.Background(), enroll.EnrollRequest{
		EnrollmentToken: "token-1",
		AgentVersion:    "test",
		RequestedHost:   "host",
		CSRPEM:          string(csr),
		BootstrapNonce:  "nonce",
	})
	if err != nil {
		t.Fatalf("enroll failed: %v", err)
	}

	pki := mtls.DefaultPKIPaths(filepath.Join(t.TempDir(), "pki"))
	if err := mtls.WritePKI(pki, mtls.EncodePrivateKeyPEM(key), []byte(resp.ClientCertificatePEM), []byte(resp.CACertificatePEM)); err != nil {
		t.Fatal(err)
	}
	mTLSClient, err := mtls.NewMutualTLSClient(pki, false)
	if err != nil {
		t.Fatal(err)
	}

	cp := controlplane.Client{BaseURL: ingestSrv.URL, HTTP: mTLSClient}
	_, err = cp.Heartbeat(context.Background(), controlplane.HeartbeatRequest{
		TenantID:      resp.TenantID,
		SiteID:        resp.SiteID,
		HostID:        resp.HostID,
		AgentID:       resp.AgentID,
		HostFacts:     hostfacts.Facts{CPUCores: 2, Arch: "amd64"},
		NetBirdStatus: netbird.Status{Connected: true},
	})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	ls := logstream.Client{BaseURL: ingestSrv.URL, HTTP: mTLSClient}
	err = ls.Stream(context.Background(), logstream.Entry{
		ExecutionID: "exec-1",
		Level:       "INFO",
		Message:     "hello",
	})
	if err != nil {
		t.Fatalf("log stream failed: %v", err)
	}

	if heartbeatCalls.Load() == 0 {
		t.Fatalf("expected heartbeat call")
	}
	if logCalls.Load() == 0 {
		t.Fatalf("expected log stream call")
	}
}

func newTestCA(t *testing.T) ([]byte, []byte, *x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject: pkix.Name{CommonName: "nkudo-test-ca"},
		NotBefore: now.Add(-1 * time.Hour),
		NotAfter:  now.Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:      true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM, cert, key
}

func newServerCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano() + 1),
		Subject: pkix.Name{CommonName: "127.0.0.1"},
		DNSNames: []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore: now.Add(-1 * time.Hour),
		NotAfter:  now.Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func signCSR(t *testing.T, csrPEM string, caCert *x509.Certificate, caKey *rsa.PrivateKey) string {
	t.Helper()
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		t.Fatal("invalid csr pem")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := csr.CheckSignature(); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano() + 2),
		Subject: csr.Subject,
		NotBefore: now.Add(-1 * time.Hour),
		NotAfter:  now.Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, csr.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
