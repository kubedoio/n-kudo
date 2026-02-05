package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/n-kudo/n-kudo-edge/pkg/controlplane"
	"github.com/n-kudo/n-kudo-edge/pkg/enroll"
	"github.com/n-kudo/n-kudo-edge/pkg/executor"
	"github.com/n-kudo/n-kudo-edge/pkg/hostfacts"
	"github.com/n-kudo/n-kudo-edge/pkg/logstream"
	"github.com/n-kudo/n-kudo-edge/pkg/mtls"
	"github.com/n-kudo/n-kudo-edge/pkg/netbird"
	"github.com/n-kudo/n-kudo-edge/pkg/providers/cloudhypervisor"
	"github.com/n-kudo/n-kudo-edge/pkg/state"
)

var version = "dev"

const (
	defaultStateDir   = "/var/lib/nkudo-edge/state"
	defaultPKIDir     = "/var/lib/nkudo-edge/pki"
	defaultRuntimeDir = "/var/lib/nkudo-edge/vms"
	defaultInterval   = 15 * time.Second
)

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
	case "version":
		fmt.Println(version)
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
  version           Print binary version`)
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

	st, err := state.Open(*stateDir)
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
		chBin      = fs.String("cloud-hypervisor-bin", "cloud-hypervisor", "Cloud Hypervisor binary path")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *planFile == "" {
		return errors.New("--plan is required")
	}

	st, err := state.Open(*stateDir)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := os.MkdirAll(*runtimeDir, 0o700); err != nil {
		return err
	}

	provider := &cloudhypervisor.Provider{Binary: *chBin, State: st, RuntimeDir: *runtimeDir}
	exec := &executor.Executor{Store: st, Provider: provider, Logs: &stdoutSink{}}

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
		controlPlane    = fs.String("control-plane", "", "Control-plane base URL (required)")
		stateDir        = fs.String("state-dir", defaultStateDir, "State directory")
		pkiDir          = fs.String("pki-dir", defaultPKIDir, "PKI directory")
		runtimeDir      = fs.String("runtime-dir", defaultRuntimeDir, "Runtime directory")
		interval        = fs.Duration("heartbeat-interval", defaultInterval, "Heartbeat interval")
		once            = fs.Bool("once", false, "Run one loop then exit")
		insecure        = fs.Bool("insecure-skip-verify", false, "Skip TLS verification (dev only)")
		netbirdSetupKey = fs.String("netbird-setup-key", "", "NetBird setup key used on first run")
		netbirdBin      = fs.String("netbird-bin", "netbird", "NetBird binary")
		chBin           = fs.String("cloud-hypervisor-bin", "cloud-hypervisor", "Cloud Hypervisor binary")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*controlPlane) == "" {
		return errors.New("--control-plane is required")
	}

	st, err := state.Open(*stateDir)
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

	cp := &controlplane.Client{BaseURL: *controlPlane, HTTP: httpClient}
	ls := &logstream.Client{BaseURL: *controlPlane, HTTP: httpClient}
	provider := &cloudhypervisor.Provider{Binary: *chBin, State: st, RuntimeDir: *runtimeDir}
	nb := netbird.Client{Binary: *netbirdBin}
	sink := &streamSink{Identity: id, Client: ls}
	exec := &executor.Executor{Store: st, Provider: provider, Logs: sink}

	if *netbirdSetupKey != "" {
		hostname, _ := os.Hostname()
		if err := nb.Join(ctx, *netbirdSetupKey, hostname); err != nil {
			log.Printf("netbird join warning: %v", err)
		} else {
			log.Printf("netbird join: ok")
		}
	}

	loop := func() error {
		facts, factsErr := hostfacts.Collect()
		if factsErr != nil {
			log.Printf("hostfacts warning: %v", factsErr)
		}
		nbStatus, nbErr := nb.Status(ctx)
		if nbErr != nil {
			log.Printf("netbird status warning: %v", nbErr)
		}
		vms, _ := st.ListMicroVMs()

		hbResp, err := cp.Heartbeat(ctx, controlplane.HeartbeatRequest{
			TenantID:      id.TenantID,
			SiteID:        id.SiteID,
			HostID:        id.HostID,
			AgentID:       id.AgentID,
			SentAt:        time.Now().UTC(),
			HostFacts:     facts,
			NetBirdStatus: nbStatus,
			MicroVMs:      vms,
		})
		if err != nil {
			return err
		}

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
			sink.Write(ctx, executor.LogEntry{ExecutionID: plan.ExecutionID, Level: "INFO", Message: "plan execution started"})
			res, runErr := exec.ExecutePlan(ctx, plan)
			if reportErr := cp.ReportPlanResult(ctx, res); reportErr != nil {
				log.Printf("plan result report warning: %v", reportErr)
			}
			if runErr != nil {
				sink.Write(ctx, executor.LogEntry{ExecutionID: plan.ExecutionID, Level: "ERROR", Message: runErr.Error()})
			} else {
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
			return nil
		case <-time.After(*interval):
		}
	}
}

func runVerifyHeartbeat(ctx context.Context, args []string) error {
	newArgs := append([]string{"--once"}, args...)
	return runService(ctx, newArgs)
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
	Client   *logstream.Client
}

func (s *streamSink) Write(ctx context.Context, entry executor.LogEntry) {
	log.Printf("execution=%s action=%s level=%s msg=%s", entry.ExecutionID, entry.ActionID, entry.Level, entry.Message)
	if s.Client == nil {
		return
	}
	err := s.Client.Stream(ctx, logstream.Entry{
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
