package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/enroll"
	"github.com/kubedoio/n-kudo/internal/edge/mtls"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

const unenrollUsage = `Usage: edge unenroll [options]

Cleanly remove agent from site. Stops running VMs, sends unenrollment
request to control plane, and clears local state.

Options:
  --state-dir string       State directory (default "/var/lib/nkudo-edge/state")
  --pki-dir string         PKI directory (default "/var/lib/nkudo-edge/pki")
  --runtime-dir string     Runtime directory (default "/var/lib/nkudo-edge/vms")
  --control-plane string   Control-plane base URL (required)
  --force                  Skip graceful VM shutdown
  --keep-dirs              Keep directories (just clear contents)
  --insecure-skip-verify   Skip TLS verification (dev only)
`

// UnenrollOptions holds the configuration for the unenroll command
type UnenrollOptions struct {
	StateDir           string
	PKIDir             string
	RuntimeDir         string
	ControlPlane       string
	Force              bool
	KeepDirs           bool
	InsecureSkipVerify bool
}

// RunUnenroll executes the unenroll command
func RunUnenroll(args []string) error {
	opts := UnenrollOptions{}
	fs := flag.NewFlagSet("unenroll", flag.ContinueOnError)
	fs.StringVar(&opts.StateDir, "state-dir", "/var/lib/nkudo-edge/state", "State directory")
	fs.StringVar(&opts.PKIDir, "pki-dir", "/var/lib/nkudo-edge/pki", "PKI directory")
	fs.StringVar(&opts.RuntimeDir, "runtime-dir", "/var/lib/nkudo-edge/vms", "Runtime directory")
	fs.StringVar(&opts.ControlPlane, "control-plane", "", "Control-plane base URL")
	fs.BoolVar(&opts.Force, "force", false, "Skip graceful VM shutdown")
	fs.BoolVar(&opts.KeepDirs, "keep-dirs", false, "Keep directories (just clear contents)")
	fs.BoolVar(&opts.InsecureSkipVerify, "insecure-skip-verify", false, "Skip TLS verification (dev only)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if opts.ControlPlane == "" {
		return errors.New("--control-plane is required")
	}

	return runUnenroll(context.Background(), opts)
}

func runUnenroll(ctx context.Context, opts UnenrollOptions) error {
	// Open state store to get identity
	st, err := state.Open(opts.StateDir)
	if err != nil {
		return fmt.Errorf("open state store: %w", err)
	}

	identity, err := st.LoadIdentity()
	if err != nil {
		st.Close()
		// If no identity, just clean up local files
		fmt.Println("No enrollment found, cleaning up local files only...")
		return cleanupLocalFiles(opts)
	}

	// List running VMs
	vms, _ := st.ListMicroVMs()
	runningVMs := []state.MicroVM{}
	for _, vm := range vms {
		if vm.Status == "running" || vm.CHPID > 0 {
			runningVMs = append(runningVMs, vm)
		}
	}

	// Stop running VMs
	if len(runningVMs) > 0 {
		fmt.Printf("Stopping running VMs... ")
		if opts.Force {
			fmt.Println("(force mode - immediate shutdown)")
		} else {
			fmt.Println("(graceful shutdown)")
		}
		stopped := 0
		for _, vm := range runningVMs {
			if err := stopVM(ctx, vm, opts.Force); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: failed to stop VM %s: %v\n", vm.ID, err)
			} else {
				stopped++
			}
		}
		fmt.Printf("  Stopped %d/%d VMs\n", stopped, len(runningVMs))
	}

	st.Close()

	// Create mTLS client for unenrollment request
	pki := mtls.DefaultPKIPaths(opts.PKIDir)
	client, err := mtls.NewMutualTLSClient(pki, opts.InsecureSkipVerify)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create mTLS client: %v\n", err)
		fmt.Println("Continuing with local cleanup...")
	} else {
		// Send unenrollment request
		fmt.Printf("Sending unenrollment request... ")
		ec := &enroll.Client{BaseURL: opts.ControlPlane, HTTP: client}
		if err := sendUnenrollRequest(ctx, ec, identity.AgentID); err != nil {
			fmt.Printf("failed (%v)\n", err)
			fmt.Println("Continuing with local cleanup...")
		} else {
			fmt.Println("done")
		}
	}

	// Clean up local files
	fmt.Printf("Clearing local state... ")
	if err := cleanupLocalFiles(opts); err != nil {
		fmt.Printf("failed (%v)\n", err)
		return err
	}
	fmt.Println("done")

	fmt.Println()
	fmt.Println("Agent successfully unenrolled from site.")
	return nil
}

func stopVM(ctx context.Context, vm state.MicroVM, force bool) error {
	if vm.CHPID <= 0 {
		return nil
	}

	// Send SIGTERM for graceful shutdown, SIGKILL for force
	sig := os.Interrupt
	if force {
		sig = os.Kill
	}

	process, err := os.FindProcess(vm.CHPID)
	if err != nil {
		return err
	}

	if err := process.Signal(sig); err != nil {
		return err
	}

	if !force {
		// Wait a bit for graceful shutdown
		select {
		case <-time.After(5 * time.Second):
			// Try SIGKILL after timeout
			_ = process.Signal(os.Kill)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func sendUnenrollRequest(ctx context.Context, client *enroll.Client, agentID string) error {
	return client.Unenroll(ctx, agentID)
}

func cleanupLocalFiles(opts UnenrollOptions) error {
	if opts.KeepDirs {
		// Just clear contents
		for _, dir := range []string{opts.StateDir, opts.PKIDir, opts.RuntimeDir} {
			if err := clearDirectory(dir); err != nil {
				return fmt.Errorf("clear %s: %w", dir, err)
			}
		}
	} else {
		// Remove entire directories
		for _, dir := range []string{opts.StateDir, opts.PKIDir, opts.RuntimeDir} {
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove %s: %w", dir, err)
			}
		}
	}
	return nil
}

func clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}

// UnenrollHelp returns the help text for the unenroll command
func UnenrollHelp() string {
	return unenrollUsage
}
