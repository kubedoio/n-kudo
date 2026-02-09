package cmd

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/mtls"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

const statusUsage = `Usage: edge status [options]

Show agent enrollment status, connection health, and certificate expiry.

Options:
  --state-dir string    State directory (default "/var/lib/nkudo-edge/state")
  --pki-dir string      PKI directory (default "/var/lib/nkudo-edge/pki")
  --control-plane       Control-plane base URL (optional, for connection check)
`

// StatusOptions holds the configuration for the status command
type StatusOptions struct {
	StateDir     string
	PKIDir       string
	ControlPlane string
}

// StatusInfo holds all the status information to display
type StatusInfo struct {
	Enrolled       bool
	TenantID       string
	SiteID         string
	HostID         string
	AgentID        string
	ControlPlane   string
	CertSerial     string
	CertExpires    time.Time
	CertValid      bool
	DaysRemaining  int
	LastHeartbeat  time.Time
	ConnectionOK   bool
}

// RunStatus executes the status command
func RunStatus(args []string) error {
	opts := StatusOptions{}
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.StringVar(&opts.StateDir, "state-dir", "/var/lib/nkudo-edge/state", "State directory")
	fs.StringVar(&opts.PKIDir, "pki-dir", "/var/lib/nkudo-edge/pki", "PKI directory")
	fs.StringVar(&opts.ControlPlane, "control-plane", "", "Control-plane base URL")
	
	if err := fs.Parse(args); err != nil {
		return err
	}

	info, err := collectStatus(opts)
	if err != nil {
		// Check if this is just "not enrolled" state
		if errors.Is(err, ErrNotEnrolled) {
			fmt.Println("Agent Status: not enrolled")
			fmt.Println("\nRun 'edge enroll' to enroll this agent with the control plane.")
			return nil
		}
		return err
	}

	printStatus(info)
	return nil
}

var ErrNotEnrolled = errors.New("agent not enrolled")

func collectStatus(opts StatusOptions) (*StatusInfo, error) {
	info := &StatusInfo{}

	// Try to open state store
	st, err := state.Open(opts.StateDir)
	if err != nil {
		return nil, fmt.Errorf("open state store: %w", err)
	}
	defer st.Close()

	// Load identity
	identity, err := st.LoadIdentity()
	if err != nil {
		if errors.Is(err, errors.New("identity not found")) {
			return nil, ErrNotEnrolled
		}
		return nil, ErrNotEnrolled
	}

	info.Enrolled = true
	info.TenantID = identity.TenantID
	info.SiteID = identity.SiteID
	info.HostID = identity.HostID
	info.AgentID = identity.AgentID

	// Read certificate info
	pki := mtls.DefaultPKIPaths(opts.PKIDir)
	certInfo, err := readCertificateInfo(pki.ClientCert)
	if err == nil {
		info.CertSerial = certInfo.Serial
		info.CertExpires = certInfo.Expires
		info.CertValid = certInfo.Valid
		info.DaysRemaining = int(time.Until(certInfo.Expires).Hours() / 24)
	}

	// Check connection if control plane is provided
	if opts.ControlPlane != "" {
		info.ControlPlane = opts.ControlPlane
		info.ConnectionOK = checkConnection(opts.ControlPlane, pki)
	}

	return info, nil
}

type certInfo struct {
	Serial  string
	Expires time.Time
	Valid   bool
}

func readCertificateInfo(certPath string) (*certInfo, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &certInfo{
		Serial:  cert.SerialNumber.String(),
		Expires: cert.NotAfter,
		Valid:   now.After(cert.NotBefore) && now.Before(cert.NotAfter),
	}, nil
}

func checkConnection(controlPlane string, pki mtls.PKIPaths) bool {
	// Try to create an mTLS client and make a simple request
	client, err := mtls.NewMutualTLSClient(pki, false)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := os.ReadFile(controlPlane + "/healthz")
	_ = req
	_ = err
	_ = client
	_ = ctx

	// For now, just return false - we'll implement proper health check later
	return false
}

func printStatus(info *StatusInfo) {
	if !info.Enrolled {
		fmt.Println("Agent Status: not enrolled")
		return
	}

	fmt.Println("Agent Status: enrolled")
	fmt.Printf("Tenant ID:    %s\n", info.TenantID)
	fmt.Printf("Site ID:      %s\n", info.SiteID)
	fmt.Printf("Host ID:      %s\n", info.HostID)
	fmt.Printf("Agent ID:     %s\n", info.AgentID)
	fmt.Println()

	fmt.Println("Certificate:")
	if info.CertSerial != "" {
		fmt.Printf("  Serial:     %s\n", info.CertSerial)
	}
	if !info.CertExpires.IsZero() {
		fmt.Printf("  Expires:    %s (%d days remaining)\n", 
			info.CertExpires.Format("2006-01-02"), info.DaysRemaining)
		validStr := "yes"
		if !info.CertValid {
			validStr = "no"
		}
		fmt.Printf("  Valid:      %s\n", validStr)
	} else {
		fmt.Println("  (certificate not found)")
	}
	fmt.Println()

	fmt.Println("Connection:")
	if info.ControlPlane != "" {
		fmt.Printf("  Control Plane: %s\n", info.ControlPlane)
	}
	if info.ConnectionOK {
		fmt.Println("  Status:        connected")
	} else {
		fmt.Println("  Status:        not connected")
	}
}

// StatusHelp returns the help text for the status command
func StatusHelp() string {
	return statusUsage
}
