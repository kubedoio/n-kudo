package controlplane

import "testing"

func TestLoadOrCreateInternalCARequirePersistent(t *testing.T) {
	t.Setenv("CA_CERT_FILE", "")
	t.Setenv("CA_KEY_FILE", "")
	if _, err := LoadOrCreateInternalCA("test-ca", true); err == nil {
		t.Fatalf("expected error when persistent pki is required without files")
	}
}

func TestGenerateServerTLSCertRequirePersistent(t *testing.T) {
	t.Setenv("SERVER_CERT_FILE", "")
	t.Setenv("SERVER_KEY_FILE", "")
	if _, err := GenerateServerTLSCert(true); err == nil {
		t.Fatalf("expected error when persistent pki is required without files")
	}
}
