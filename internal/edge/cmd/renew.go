package cmd

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/enroll"
	"github.com/kubedoio/n-kudo/internal/edge/mtls"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

const renewUsage = `Usage: edge renew [options]

Manual certificate renewal. Generates a new CSR, sends it to the control
plane with the refresh token, and stores the new certificate.

Options:
  --state-dir string       State directory (default "/var/lib/nkudo-edge/state")
  --pki-dir string         PKI directory (default "/var/lib/nkudo-edge/pki")
  --control-plane string   Control-plane base URL (required)
  --insecure-skip-verify   Skip TLS verification (dev only)
`

// RenewOptions holds the configuration for the renew command
type RenewOptions struct {
	StateDir           string
	PKIDir             string
	ControlPlane       string
	InsecureSkipVerify bool
}

// RunRenew executes the renew command
func RunRenew(args []string) error {
	opts := RenewOptions{}
	fs := flag.NewFlagSet("renew", flag.ContinueOnError)
	fs.StringVar(&opts.StateDir, "state-dir", "/var/lib/nkudo-edge/state", "State directory")
	fs.StringVar(&opts.PKIDir, "pki-dir", "/var/lib/nkudo-edge/pki", "PKI directory")
	fs.StringVar(&opts.ControlPlane, "control-plane", "", "Control-plane base URL")
	fs.BoolVar(&opts.InsecureSkipVerify, "insecure-skip-verify", false, "Skip TLS verification (dev only)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if opts.ControlPlane == "" {
		return errors.New("--control-plane is required")
	}

	return runRenew(context.Background(), opts)
}

func runRenew(ctx context.Context, opts RenewOptions) error {
	// Open state store
	st, err := state.Open(opts.StateDir)
	if err != nil {
		return fmt.Errorf("open state store: %w", err)
	}
	defer st.Close()

	// Load identity
	identity, err := st.LoadIdentity()
	if err != nil {
		return fmt.Errorf("load identity (run enroll first): %w", err)
	}

	// Read current certificate for display
	pki := mtls.DefaultPKIPaths(opts.PKIDir)
	currentExpiry, _ := getCertExpiry(pki.ClientCert)
	if !currentExpiry.IsZero() {
		daysRemaining := int(time.Until(currentExpiry).Hours() / 24)
		fmt.Printf("Current certificate expires: %s (%d days remaining)\n", 
			currentExpiry.Format("2006-01-02"), daysRemaining)
	}

	// Generate new private key and CSR
	fmt.Printf("Generating new CSR... ")
	key, err := mtls.GeneratePrivateKey()
	if err != nil {
		fmt.Println("failed")
		return fmt.Errorf("generate private key: %w", err)
	}

	hostname, _ := os.Hostname()
	csrPEM, err := mtls.GenerateCSRPEM(key, hostname)
	if err != nil {
		fmt.Println("failed")
		return fmt.Errorf("generate CSR: %w", err)
	}
	fmt.Println("done")

	// Create mTLS client using current credentials
	client, err := mtls.NewMutualTLSClient(pki, opts.InsecureSkipVerify)
	if err != nil {
		return fmt.Errorf("create mTLS client: %w", err)
	}

	// Request certificate renewal
	fmt.Printf("Requesting certificate renewal... ")
	ec := &enroll.Client{BaseURL: opts.ControlPlane, HTTP: client}
	resp, err := requestRenewal(ctx, ec, identity.AgentID, string(csrPEM), identity.RefreshToken)
	if err != nil {
		fmt.Println("failed")
		return fmt.Errorf("renewal request: %w", err)
	}
	fmt.Println("done")

	// Parse new expiry
	newExpiry, _ := time.Parse(time.RFC3339, resp.ExpiresAt)
	if !newExpiry.IsZero() {
		fmt.Printf("New certificate received (expires: %s)\n", newExpiry.Format("2006-01-02"))
	}

	// Store new certificate
	keyPEM := mtls.EncodePrivateKeyPEM(key)
	if err := mtls.WritePKI(pki, keyPEM, []byte(resp.ClientCertificatePEM), nil); err != nil {
		return fmt.Errorf("write PKI: %w", err)
	}

	// Update refresh token if provided
	if resp.RefreshToken != "" {
		identity.RefreshToken = resp.RefreshToken
		if err := st.SaveIdentity(identity); err != nil {
			return fmt.Errorf("save identity: %w", err)
		}
	}

	// Test the new certificate
	fmt.Printf("Testing new certificate... ")
	newClient, err := mtls.NewMutualTLSClient(pki, opts.InsecureSkipVerify)
	if err != nil {
		fmt.Println("failed")
		return fmt.Errorf("create client with new cert: %w", err)
	}

	testClient := &enroll.Client{BaseURL: opts.ControlPlane, HTTP: newClient}
	if err := testNewCertificate(ctx, testClient); err != nil {
		fmt.Println("failed")
		return fmt.Errorf("test new certificate: %w", err)
	}
	fmt.Println("done")

	fmt.Println()
	fmt.Println("Certificate successfully renewed!")
	return nil
}

func getCertExpiry(certPath string) (time.Time, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return time.Time{}, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}, errors.New("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}

	return cert.NotAfter, nil
}

func requestRenewal(ctx context.Context, client *enroll.Client, agentID, csrPEM, refreshToken string) (*enroll.RenewResponse, error) {
	return client.RenewCertificate(ctx, agentID, csrPEM, refreshToken)
}

func testNewCertificate(ctx context.Context, client *enroll.Client) error {
	// Try to fetch plans as a test - this validates the certificate works
	_, err := client.FetchPlans(ctx, "", "")
	// We expect an error about missing site/agent ID, but connection should work
	if err != nil {
		// Check if it's a connection error vs API error
		errStr := err.Error()
		if contains(errStr, "connection", "tls", "certificate") {
			return err
		}
	}
	return nil
}

func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// GenerateCSRPEMForRenewal generates a new CSR using the existing private key or creates a new one
func GenerateCSRPEMForRenewal(existingKey *rsa.PrivateKey, hostname string) ([]byte, *rsa.PrivateKey, error) {
	key := existingKey
	if key == nil {
		var err error
		key, err = mtls.GeneratePrivateKey()
		if err != nil {
			return nil, nil, err
		}
	}

	csrPEM, err := mtls.GenerateCSRPEM(key, hostname)
	if err != nil {
		return nil, nil, err
	}

	return csrPEM, key, nil
}

// RenewHelp returns the help text for the renew command
func RenewHelp() string {
	return renewUsage
}
