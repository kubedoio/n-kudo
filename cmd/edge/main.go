package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	edgecmd "github.com/kubedoio/n-kudo/internal/edge/cmd"
	"github.com/kubedoio/n-kudo/internal/edge/enroll"
	"github.com/kubedoio/n-kudo/internal/edge/executor"
	"github.com/kubedoio/n-kudo/internal/edge/hostfacts"
	"github.com/kubedoio/n-kudo/internal/edge/logger"
	"github.com/kubedoio/n-kudo/internal/edge/metrics"
	"github.com/kubedoio/n-kudo/internal/edge/mtls"
	"github.com/kubedoio/n-kudo/internal/edge/netbird"
	"github.com/kubedoio/n-kudo/internal/edge/providers/cloudhypervisor"
	"github.com/kubedoio/n-kudo/internal/edge/providers/firecracker"
	"github.com/kubedoio/n-kudo/internal/edge/securestate"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

var version = "dev"

const (
	defaultStateDir   = "/var/lib/nkudo-edge/state"
	defaultPKIDir     = "/var/lib/nkudo-edge/pki"
	defaultRuntimeDir = "/var/lib/nkudo-edge/vms"
	defaultInterval   = 15 * time.Second

	providerCloudHypervisor = "cloud-hypervisor"
	providerFirecracker     = "firecracker"
	providerAuto            = "auto"
)

// StateStore is the interface for state storage, satisfied by both
// state.Store and securestate.Store
type StateStore interface {
	Close() error
	SaveIdentity(identity state.Identity) error
	LoadIdentity() (state.Identity, error)
	UpsertMicroVM(vm state.MicroVM) error
	GetMicroVM(vmID string) (state.MicroVM, bool, error)
	DeleteMicroVM(vmID string) error
	ListMicroVMs() ([]state.MicroVM, error)
	GetActionRecord(actionID string) (state.ActionRecord, bool, error)
	PutActionRecord(record state.ActionRecord) error
}

// openState opens the state store, using securestate if NKUDO_STATE_KEY is set,
// otherwise falling back to the standard unencrypted state store.
func openState(dir string) (StateStore, error) {
	// Try secure state first - it will use NKUDO_STATE_KEY if available
	store, err := securestate.Open(dir)
	if err == nil {
		return store, nil
	}

	// If the error indicates encrypted file exists but no key, fail
	if errors.Is(err, securestate.ErrInvalidKey) ||
		(err != nil && containsString(err.Error(), "encrypted state file exists")) {
		return nil, err
	}

	// Fall back to unencrypted state store
	log.Println("[main] Falling back to unencrypted state store")
	return state.Open(dir)
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// providerSelection holds the selected provider info
type providerSelection struct {
	Name     string
	Binary   string
	Provider executor.MicroVMProvider
}

// autoDetectProvider detects which VM provider is available.
// It prefers cloud-hypervisor over firecracker if both are available.
func autoDetectProvider() (string, string) {
	// Check cloud-hypervisor first
	if _, err := exec.LookPath("cloud-hypervisor"); err == nil {
		return providerCloudHypervisor, "cloud-hypervisor"
	}
	// Fall back to firecracker
	if _, err := exec.LookPath("firecracker"); err == nil {
		return providerFirecracker, "firecracker"
	}
	// Default to cloud-hypervisor even if not found (will fail later with clear error)
	return providerCloudHypervisor, "cloud-hypervisor"
}

// selectProvider creates the appropriate provider based on configuration.
func selectProvider(providerName, chBin, fcBin string, st StateStore, runtimeDir string) (*providerSelection, error) {
	// Auto-detect if needed
	if providerName == providerAuto || providerName == "" {
		detected, bin := autoDetectProvider()
		log.Printf("[main] Auto-detected provider: %s (binary: %s)", detected, bin)
		providerName = detected
		if detected == providerCloudHypervisor && chBin == "" {
			chBin = bin
		} else if detected == providerFirecracker && fcBin == "" {
			fcBin = bin
		}
	}

	switch providerName {
	case providerCloudHypervisor:
		if chBin == "" {
			chBin = "cloud-hypervisor"
		}
		provider := &cloudhypervisor.Provider{
			Binary:     chBin,
			State:      st,
			RuntimeDir: runtimeDir,
		}
		return &providerSelection{
			Name:     providerCloudHypervisor,
			Binary:   chBin,
			Provider: provider,
		}, nil
	case providerFirecracker:
		if fcBin == "" {
			fcBin = "firecracker"
		}
		provider := &firecracker.Provider{
			Binary:     fcBin,
			State:      st,
			RuntimeDir: runtimeDir,
		}
		return &providerSelection{
			Name:     providerFirecracker,
			Binary:   fcBin,
			Provider: provider,
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (valid: %s, %s, %s)",
			providerName, providerCloudHypervisor, providerFirecracker, providerAuto)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch os.Args[1] {
	case "enroll":
		err = runEnroll(ctx, os.Args[2:])
	case "run":
		err = runService(ctx, os.Args[2:])
	case "hostfacts":
		err = runHostFacts()
	case "apply":
		err = runApply(ctx, os.Args[2:])
	case "verify-heartbeat":
		err = runVerifyHeartbeat(ctx, os.Args[2:])
	case "status":
		err = runStatus(os.Args[2:])
	case "check":
		os.Exit(edgecmd.RunCheck(os.Args[2:]))
	case "unenroll":
		err = edgecmd.RunUnenroll(os.Args[2:])
	case "renew":
		err = edgecmd.RunRenew(os.Args[2:])
	case "version":
		fmt.Println(version)
	case "--help", "-h", "help":
		usage()
	default:
		usage()
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`nkudo-edge commands:
  enroll            Enroll with one-time token and persist mTLS credentials
  run               Start heartbeat loop and execute plans
  hostfacts         Print host facts JSON
  apply             Execute a local plan JSON file
  verify-heartbeat  Send a single heartbeat
  status            Show agent enrollment status and certificate info
  check             Pre-flight check for requirements
  unenroll          Cleanly remove agent from site
  renew             Manual certificate renewal
  version           Print binary version

Use "edge <command> --help" for more information about a command.`)
}

func runEnroll(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("enroll", flag.ContinueOnError)
	var (
		controlPlane = fs.String("control-plane", "", "Control-plane base URL (required)")
		tokenFlag    = fs.String("token", "", "Enrollment token")
		tokenFile    = fs.String("token-file", "", "Path to file containing enrollment token")
		stateDir     = fs.String("state-dir", defaultStateDir, "State directory")
		pkiDir       = fs.String("pki-dir", defaultPKIDir, "PKI directory")
		hostname     = fs.String("hostname", "", "Requested hostname")
		caFile       = fs.String("ca-file", "", "Bootstrap CA certificate PEM path")
		insecure     = fs.Bool("insecure-skip-verify", false, "Skip TLS verification (dev only)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*controlPlane) == "" {
		return errors.New("--control-plane is required")
	}

	token, err := enroll.ResolveToken(enroll.TokenSource{CLIValue: *tokenFlag, EnvName: "NKUDO_ENROLL_TOKEN", FilePath: *tokenFile})
	if err != nil {
		return err
	}

	key, err := mtls.GeneratePrivateKey()
	if err != nil {
		return err
	}
	resolvedHostname := *hostname
	if resolvedHostname == "" {
		resolvedHostname, _ = os.Hostname()
	}
	csrPEM, err := mtls.GenerateCSRPEM(key, resolvedHostname)
	if err != nil {
		return err
	}
	keyPEM := mtls.EncodePrivateKeyPEM(key)

	var bootstrapCA []byte
	if *caFile != "" {
		bootstrapCA, err = os.ReadFile(*caFile)
		if err != nil {
			return fmt.Errorf("read --ca-file: %w", err)
		}
	}
	client, err := mtls.NewBootstrapTLSClient(bootstrapCA, *insecure)
	if err != nil {
		return err
	}
	ec := enroll.Client{BaseURL: *controlPlane, HTTP: client}
	resp, err := ec.Enroll(ctx, enroll.EnrollRequest{
		EnrollmentToken: token,
		AgentVersion:    version,
		RequestedHost:   resolvedHostname,
		CSRPEM:          string(csrPEM),
		Fingerprint:     enroll.BuildFingerprint(),
		BootstrapNonce:  enroll.NewNonce(),
		Labels:          map[string]string{"arch": runtimeArch(), "os": runtimeOS()},
	})
	if err != nil {
		return err
	}

	paths := mtls.DefaultPKIPaths(*pkiDir)
	if err := mtls.WritePKI(paths, keyPEM, []byte(resp.ClientCertificatePEM), []byte(resp.CACertificatePEM)); err != nil {
		return err
	}

	st, err := openState(*stateDir)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.SaveIdentity(state.Identity{
		TenantID:     resp.TenantID,
		SiteID:       resp.SiteID,
		HostID:       resp.HostID,
		AgentID:      resp.AgentID,
		RefreshToken: resp.RefreshToken,
	}); err != nil {
		return err
	}

	fmt.Printf("enrolled agent_id=%s site_id=%s host_id=%s\n", resp.AgentID, resp.SiteID, resp.HostID)
	fmt.Printf("pki written under %s\n", *pkiDir)
	return nil
}

func runHostFacts() error {
	facts, err := hostfacts.Collect()
	if err != nil {
		return err
	}
	b, err := facts.JSON()
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func runApply(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	var (
		planFile   = fs.String("plan", "", "Path to plan JSON")
		stateDir   = fs.String("state-dir", defaultStateDir, "State directory")
		runtimeDir = fs.String("runtime-dir", defaultRuntimeDir, "Runtime directory")
		provider   = fs.String("provider", providerAuto, "VM provider: cloud-hypervisor, firecracker, auto")
		chBin      = fs.String("cloud-hypervisor-bin", "cloud-hypervisor", "Cloud Hypervisor binary path")
		fcBin      = fs.String("firecracker-bin", "firecracker", "Firecracker binary path")
		logFormat  = fs.String("log-format", "text", "Log format: json or text")
		logLevel   = fs.String("log-level", "info", "Log level: debug, info, warn, error")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *planFile == "" {
		return errors.New("--plan is required")
	}

	// Initialize structured logger
	logger.Init(*logFormat, *logLevel)

	st, err := openState(*stateDir)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := os.MkdirAll(*runtimeDir, 0o700); err != nil {
		return err
	}

	sel, err := selectProvider(*provider, *chBin, *fcBin, st, *runtimeDir)
	if err != nil {
		return err
	}
	logger.WithFields(map[string]interface{}{
		"provider": sel.Name,
		"binary":   sel.Binary,
	}).Info("Using VM provider")

	exec := &executor.Executor{Store: st, Provider: sel.Provider, Logs: &stdoutSink{}}

	plan, err := readPlan(*planFile)
	if err != nil {
		return err
	}
	res, err := exec.ExecutePlan(ctx, plan)
	payload, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(payload))
	return err
}

func runService(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var (
		controlPlane        = fs.String("control-plane", "", "Control-plane base URL (required)")
		stateDir            = fs.String("state-dir", defaultStateDir, "State directory")
		pkiDir              = fs.String("pki-dir", defaultPKIDir, "PKI directory")
		runtimeDir          = fs.String("runtime-dir", defaultRuntimeDir, "Runtime directory")
		interval            = fs.Duration("heartbeat-interval", defaultInterval, "Heartbeat interval")
		once                = fs.Bool("once", false, "Run one loop then exit")
		insecure            = fs.Bool("insecure-skip-verify", false, "Skip TLS verification (dev only)")
		netbirdSetupKey     = fs.String("netbird-setup-key", "", "NetBird setup key used on first run")
		netbirdBin          = fs.String("netbird-bin", "netbird", "NetBird binary")
		netbirdEnabled      = fs.Bool("netbird-enabled", true, "Enable NetBird connectivity checks")
		netbirdAutoJoin     = fs.Bool("netbird-auto-join", true, "Auto-join NetBird when setup key is provided")
		netbirdHostname     = fs.String("netbird-hostname", "", "NetBird hostname override (defaults to OS hostname)")
		netbirdInstall      = fs.String("netbird-install-cmd", "", "Optional install command when netbird CLI is missing")
		netbirdService      = fs.Bool("netbird-require-service", true, "Require NetBird service/process to be running")
		netbirdProbeType    = fs.String("netbird-probe-type", "http", "NetBird probe type: http|ping")
		netbirdProbeTarget  = fs.String("netbird-probe-target", "", "NetBird mesh endpoint for readiness checks")
		netbirdProbeTimeout = fs.Duration("netbird-probe-timeout", 5*time.Second, "NetBird probe timeout")
		netbirdProbeHTTPMin = fs.Int("netbird-probe-http-min", 200, "NetBird HTTP probe minimum status code")
		netbirdProbeHTTPMax = fs.Int("netbird-probe-http-max", 399, "NetBird HTTP probe maximum status code")
		providerName        = fs.String("provider", providerAuto, "VM provider: cloud-hypervisor, firecracker, auto")
		chBin               = fs.String("cloud-hypervisor-bin", "cloud-hypervisor", "Cloud Hypervisor binary")
		fcBin               = fs.String("firecracker-bin", "firecracker", "Firecracker binary")
		metricsAddr         = fs.String("metrics-addr", ":9090", "Metrics server address")
		logFormat           = fs.String("log-format", "text", "Log format: json or text")
		logLevel            = fs.String("log-level", "info", "Log level: debug, info, warn, error")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*controlPlane) == "" {
		return errors.New("--control-plane is required")
	}

	// Initialize structured logger
	logger.Init(*logFormat, *logLevel)

	// Start metrics server
	go func() {
		logger.WithFields(map[string]interface{}{
			"address": *metricsAddr,
		}).Info("Starting metrics server")
		if err := metrics.StartServer(*metricsAddr); err != nil {
			logger.Errorf("Metrics server error: %v", err)
		}
	}()

	st, err := openState(*stateDir)
	if err != nil {
		return err
	}
	defer st.Close()
	id, err := st.LoadIdentity()
	if err != nil {
		return fmt.Errorf("load identity (run enroll first): %w", err)
	}
	if err := os.MkdirAll(*runtimeDir, 0o700); err != nil {
		return err
	}

	pki := mtls.DefaultPKIPaths(*pkiDir)
	httpClient, err := mtls.NewMutualTLSClient(pki, *insecure)
	if err != nil {
		return err
	}

	cp := &enroll.Client{BaseURL: *controlPlane, HTTP: httpClient}
	sel, err := selectProvider(*providerName, *chBin, *fcBin, st, *runtimeDir)
	if err != nil {
		return err
	}
	logger.WithFields(map[string]interface{}{
		"provider": sel.Name,
		"binary":   sel.Binary,
	}).Info("Using VM provider")

	nb := netbird.Client{Binary: *netbirdBin}
	sink := &streamSink{Identity: id, Client: cp}
	exec := &executor.Executor{Store: st, Provider: sel.Provider, Logs: sink}

	// Start certificate rotator
	certRotator := mtls.NewCertRotator(pki, id, cp)
	if err := certRotator.Start(ctx); err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Failed to start certificate rotator")
	}
	defer certRotator.Stop()

	resolvedNBHostname := strings.TrimSpace(*netbirdHostname)
	if resolvedNBHostname == "" {
		resolvedNBHostname, _ = os.Hostname()
	}
	netbirdSetupKeyValue := strings.TrimSpace(*netbirdSetupKey)
	netbirdInstallCommand := splitCommand(*netbirdInstall)

	loop := func() error {
		hbStart := time.Now()

		facts, factsErr := hostfacts.Collect()
		if factsErr != nil {
			logger.WithFields(map[string]interface{}{
				"error": factsErr.Error(),
			}).Warn("hostfacts collection warning")
		} else {
			// Update host metrics using available fields
			memoryUsed := float64(facts.MemoryTotal - facts.MemoryFree)
			metrics.HostMemoryUsageBytes.Set(memoryUsed)
			// CPU percent not directly available, use CPU cores as a proxy
			metrics.HostCPUUsagePercent.Set(float64(facts.CPUCores))
		}

		nbSnapshot, nbErr := nb.Evaluate(ctx, netbird.Config{
			Enabled:        *netbirdEnabled,
			AutoJoin:       *netbirdAutoJoin,
			SetupKey:       netbirdSetupKeyValue,
			Hostname:       resolvedNBHostname,
			RequireService: *netbirdService,
			InstallCommand: netbirdInstallCommand,
			Probe: netbird.ProbeConfig{
				Type:          netbird.ProbeType(strings.ToLower(strings.TrimSpace(*netbirdProbeType))),
				Target:        strings.TrimSpace(*netbirdProbeTarget),
				Timeout:       *netbirdProbeTimeout,
				HTTPStatusMin: *netbirdProbeHTTPMin,
				HTTPStatusMax: *netbirdProbeHTTPMax,
			},
		})
		if netbirdSetupKeyValue != "" {
			netbirdSetupKeyValue = ""
		}
		if nbErr != nil {
			logger.WithFields(map[string]interface{}{
				"error": nbErr.Error(),
			}).Warn("netbird evaluation warning")
		}
		nbStatus := nbSnapshot.Peer
		nbStatus.Connected = nbSnapshot.ControlPlaneConnected()
		nbStatus.State = string(nbSnapshot.State)
		nbStatus.Reason = nbSnapshot.Reason

		// List VMs and update metrics
		vms, _ := st.ListMicroVMs()
		updateVMMetrics(vms)

		hbResp, err := cp.Heartbeat(ctx, enroll.HeartbeatRequest{
			TenantID:      id.TenantID,
			SiteID:        id.SiteID,
			HostID:        id.HostID,
			AgentID:       id.AgentID,
			SentAt:        time.Now().UTC(),
			HostFacts:     facts,
			NetBirdStatus: nbStatus,
			MicroVMs:      vms,
		})

		// Record heartbeat metrics
		hbDuration := time.Since(hbStart)
		metrics.HeartbeatDuration.Observe(hbDuration.Seconds())

		if err != nil {
			metrics.HeartbeatFailures.Inc()
			logger.WithFields(map[string]interface{}{
				"duration_ms": hbDuration.Milliseconds(),
				"error":       err.Error(),
			}).Error("Heartbeat failed")
			return err
		}

		metrics.HeartbeatsSent.Inc()
		logger.WithFields(map[string]interface{}{
			"duration_ms": hbDuration.Milliseconds(),
		}).Debug("Heartbeat sent successfully")

		plans := hbResp.PendingPlans
		if len(plans) == 0 {
			if nextPlans, e := cp.FetchPlans(ctx, id.SiteID, id.AgentID); e == nil {
				plans = nextPlans
			}
		}

		for _, plan := range plans {
			if strings.TrimSpace(plan.ExecutionID) == "" {
				plan.ExecutionID = fmt.Sprintf("exec-%d", time.Now().UTC().UnixNano())
			}
			logger.WithFields(map[string]interface{}{
				"execution_id": plan.ExecutionID,
				"plan_id":      plan.PlanID,
			}).Info("Plan execution started")
			sink.Write(ctx, executor.LogEntry{ExecutionID: plan.ExecutionID, Level: "INFO", Message: "plan execution started"})
			res, runErr := exec.ExecutePlan(ctx, plan)
			if reportErr := cp.ReportPlanResult(ctx, res); reportErr != nil {
				logger.WithFields(map[string]interface{}{
					"error": reportErr.Error(),
				}).Warn("plan result report warning")
			}
			if runErr != nil {
				logger.WithFields(map[string]interface{}{
					"execution_id": plan.ExecutionID,
					"error":        runErr.Error(),
				}).Error("Plan execution failed")
				sink.Write(ctx, executor.LogEntry{ExecutionID: plan.ExecutionID, Level: "ERROR", Message: runErr.Error()})
			} else {
				logger.WithFields(map[string]interface{}{
					"execution_id": plan.ExecutionID,
				}).Info("Plan execution finished")
				sink.Write(ctx, executor.LogEntry{ExecutionID: plan.ExecutionID, Level: "INFO", Message: "plan execution finished"})
			}
		}

		if hbResp.NextHeartbeatSeconds > 0 {
			*interval = time.Duration(hbResp.NextHeartbeatSeconds) * time.Second
		}
		return nil
	}

	for {
		if err := loop(); err != nil {
			log.Printf("loop error: %v", err)
		}
		if *once {
			return nil
		}
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received, starting graceful shutdown...")

			// Stop all running VMs gracefully
			logger.Info("stopping all running VMs...")
			if err := stopAllVMsGracefully(context.Background(), st, sel.Provider); err != nil {
				logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Warn("error stopping VMs during shutdown")
			}

			// Send final heartbeat with shutdown status
			logger.Info("sending final heartbeat...")
			if err := sendFinalHeartbeat(context.Background(), cp, id, st, netbird.Status{Connected: false, State: "shutdown", Reason: "agent_shutdown"}); err != nil {
				logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Warn("failed to send final heartbeat")
			}

			logger.Info("graceful shutdown complete")
			return nil
		case <-time.After(*interval):
		}
	}
}

func runVerifyHeartbeat(ctx context.Context, args []string) error {
	newArgs := append([]string{"--once"}, args...)
	return runService(ctx, newArgs)
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	var (
		pkiDir   = fs.String("pki-dir", defaultPKIDir, "PKI directory")
		stateDir = fs.String("state-dir", defaultStateDir, "State directory")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load identity
	st, err := openState(*stateDir)
	if err != nil {
		return fmt.Errorf("open state: %w", err)
	}
	defer st.Close()

	identity, err := st.LoadIdentity()
	if err != nil {
		return fmt.Errorf("load identity: %w (run enroll first)", err)
	}

	fmt.Printf("Agent ID:    %s\n", identity.AgentID)
	fmt.Printf("Tenant ID:   %s\n", identity.TenantID)
	fmt.Printf("Site ID:     %s\n", identity.SiteID)
	fmt.Printf("Host ID:     %s\n", identity.HostID)
	fmt.Printf("Saved At:    %s\n", identity.SavedAt.Format(time.RFC3339))

	// Load certificate info
	pki := mtls.DefaultPKIPaths(*pkiDir)
	cert, err := mtls.LoadCertificate(pki.ClientCert)
	if err != nil {
		fmt.Printf("\nCertificate: not available (%v)\n", err)
	} else {
		now := time.Now().UTC()
		remaining := cert.NotAfter.Sub(now)
		totalLifetime := cert.NotAfter.Sub(cert.NotBefore)
		percentRemaining := float64(remaining) / float64(totalLifetime) * 100

		fmt.Printf("\nCertificate Information:\n")
		fmt.Printf("  Subject:     %s\n", cert.Subject.CommonName)
		fmt.Printf("  Issuer:      %s\n", cert.Issuer.CommonName)
		fmt.Printf("  Serial:      %s\n", cert.SerialNumber.String())
		fmt.Printf("  Not Before:  %s\n", cert.NotBefore.Format(time.RFC3339))
		fmt.Printf("  Not After:   %s\n", cert.NotAfter.Format(time.RFC3339))
		fmt.Printf("  Remaining:   %s (%.1f%%)\n", remaining.Round(time.Second), percentRemaining)

		// Check if rotation is needed
		if remaining <= 0 {
			fmt.Printf("  Status:      EXPIRED\n")
		} else if remaining < 6*time.Hour || percentRemaining < 20 {
			fmt.Printf("  Status:      ROTATION RECOMMENDED\n")
		} else {
			fmt.Printf("  Status:      OK\n")
		}
	}

	return nil
}

func readPlan(path string) (executor.Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return executor.Plan{}, err
	}
	var p executor.Plan
	if err := json.Unmarshal(b, &p); err != nil {
		return executor.Plan{}, err
	}
	if p.ExecutionID == "" {
		p.ExecutionID = fmt.Sprintf("exec-%d", time.Now().UTC().UnixNano())
	}
	for i, a := range p.Actions {
		if strings.TrimSpace(a.ActionID) == "" {
			p.Actions[i].ActionID = fmt.Sprintf("action-%d", i+1)
		}
	}
	return p, nil
}

func runtimeOS() string {
	facts, err := hostfacts.Collect()
	if err != nil {
		return "linux"
	}
	return facts.OSType
}

func runtimeArch() string {
	facts, err := hostfacts.Collect()
	if err != nil {
		return "unknown"
	}
	return facts.Arch
}

type stdoutSink struct{}

func (s *stdoutSink) Write(_ context.Context, entry executor.LogEntry) {
	log.Printf("execution=%s action=%s level=%s msg=%s", entry.ExecutionID, entry.ActionID, entry.Level, entry.Message)
}

type streamSink struct {
	Identity state.Identity
	Client   *enroll.Client
}

func (s *streamSink) Write(ctx context.Context, entry executor.LogEntry) {
	log.Printf("execution=%s action=%s level=%s msg=%s", entry.ExecutionID, entry.ActionID, entry.Level, entry.Message)
	if s.Client == nil {
		return
	}
	err := s.Client.StreamLog(ctx, enroll.LogEntry{
		TenantID:    s.Identity.TenantID,
		SiteID:      s.Identity.SiteID,
		AgentID:     s.Identity.AgentID,
		ExecutionID: entry.ExecutionID,
		ActionID:    entry.ActionID,
		Level:       strings.ToUpper(entry.Level),
		Message:     entry.Message,
	})
	if err != nil {
		log.Printf("log stream warning: %v", err)
	}
}

func ensurePath(base, child string) string {
	if child == "" {
		return base
	}
	if filepath.IsAbs(child) {
		return child
	}
	return filepath.Join(base, child)
}

func splitCommand(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return strings.Fields(trimmed)
}

func stopAllVMsGracefully(ctx context.Context, st StateStore, provider executor.MicroVMProvider) error {
	vms, err := st.ListMicroVMs()
	if err != nil {
		return fmt.Errorf("list VMs: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(vms))

	for _, vm := range vms {
		if vm.Status != "running" {
			continue
		}
		wg.Add(1)
		go func(vmID string) {
			defer wg.Done()
			logger.WithFields(map[string]interface{}{
				"vm_id": vmID,
			}).Info("stopping VM gracefully")
			if err := provider.Stop(ctx, vmID); err != nil {
				errCh <- fmt.Errorf("stop VM %s: %w", vmID, err)
			}
		}(vm.ID)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping VMs: %v", errs)
	}
	return nil
}

func sendFinalHeartbeat(ctx context.Context, cp *enroll.Client, id state.Identity, st StateStore, lastNBStatus netbird.Status) error {
	vms, _ := st.ListMicroVMs()

	hbReq := enroll.HeartbeatRequest{
		TenantID:  id.TenantID,
		SiteID:    id.SiteID,
		HostID:    id.HostID,
		AgentID:   id.AgentID,
		SentAt:    time.Now().UTC(),
		HostFacts: hostfacts.Facts{OSType: runtimeOS(), Arch: runtimeArch()},
		NetBirdStatus: netbird.Status{
			Connected: lastNBStatus.Connected,
			State:     string(lastNBStatus.State),
			Reason:    "agent_shutdown",
		},
		MicroVMs: vms,
		Shutdown: true,
	}

	_, err := cp.Heartbeat(ctx, hbReq)
	return err
}

func updateVMMetrics(vms []state.MicroVM) {
	runningCount := 0
	stoppedCount := 0

	for _, vm := range vms {
		switch vm.Status {
		case "running":
			runningCount++
		case "stopped":
			stoppedCount++
		}

		// Update per-VM metrics with available fields
		// Note: MicroVM struct doesn't have Cores/Memory fields in current implementation
		// We register the VM presence for tracking with ID (not VMID)
		metrics.VMCPUCores.WithLabelValues(vm.ID, vm.Name).Set(0)
		metrics.VMMemoryBytes.WithLabelValues(vm.ID, vm.Name).Set(0)
	}

	metrics.VMsTotal.WithLabelValues("running").Set(float64(runningCount))
	metrics.VMsTotal.WithLabelValues("stopped").Set(float64(stoppedCount))
}
