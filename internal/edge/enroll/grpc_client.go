package enroll

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	controlplanev1 "github.com/kubedoio/n-kudo/api/proto/controlplane/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// GRPCClient is a gRPC client for the control plane
type GRPCClient struct {
	conn   *grpc.ClientConn
	enroll controlplanev1.EnrollmentServiceClient
	agent  controlplanev1.AgentControlServiceClient
	tenant controlplanev1.TenantControlServiceClient
}

// GRPCEnrollRequest represents an enrollment request for gRPC
type GRPCEnrollRequest struct {
	EnrollmentToken   string
	AgentVersion      string
	Hostname          string
	CSRPEM            string
	Labels            map[string]string
	BootstrapNonce    string
}

// GRPCEnrollResponse represents an enrollment response from gRPC
type GRPCEnrollResponse struct {
	TenantID                 string
	SiteID                   string
	HostID                   string
	AgentID                  string
	ClientCertificatePEM     string
	CACertificatePEM         string
	RefreshToken             string
	HeartbeatEndpoint        string
	HeartbeatIntervalSeconds int
}

// GRPCHeartbeatRequest represents a heartbeat request for gRPC
type GRPCHeartbeatRequest struct {
	TenantID         string
	SiteID           string
	HostID           string
	AgentID          string
	HeartbeatSeq     int64
	AgentVersion     string
	OS               string
	Arch             string
	KernelVersion    string
	Hostname         string
	CPUCoresTotal    int
	MemoryBytesTotal int64
	StorageBytesTotal int64
	KVMAvailable     bool
	CloudHypervisorAvailable bool
	MicroVMs         []GRPCMicroVMStatus
}

// GRPCMicroVMStatus represents a microVM status in heartbeat
type GRPCMicroVMStatus struct {
	ID        string
	Name      string
	State     string
	VCPUCount int
	MemoryMiB int64
}

// GRPCHeartbeatResponse represents a heartbeat response
type GRPCHeartbeatResponse struct {
	NextHeartbeatSeconds int
	PendingPlans         []GRPCPlan
	RotateCertificate    bool
}

// GRPCPlan represents a plan in heartbeat response
type GRPCPlan struct {
	PlanID      string
	ExecutionID string
	Actions     []GRPCPlanAction
}

// GRPCPlanAction represents a plan action
type GRPCPlanAction struct {
	ActionID string
	Type     string
	Params   map[string]interface{}
	Timeout  int
}

// NewGRPCClient creates a new gRPC client for the control plane
func NewGRPCClient(target string, tlsConfig *tls.Config) (*GRPCClient, error) {
	var opts []grpc.DialOption

	// Configure keepalive
	kpParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}
	opts = append(opts, grpc.WithKeepaliveParams(kpParams))

	// Configure TLS
	if tlsConfig != nil {
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// Set default timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to control plane: %w", err)
	}

	return &GRPCClient{
		conn:   conn,
		enroll: controlplanev1.NewEnrollmentServiceClient(conn),
		agent:  controlplanev1.NewAgentControlServiceClient(conn),
		tenant: controlplanev1.NewTenantControlServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Enroll performs agent enrollment
func (c *GRPCClient) Enroll(ctx context.Context, req GRPCEnrollRequest) (*GRPCEnrollResponse, error) {
	grpcReq := &controlplanev1.EnrollRequest{
		EnrollmentToken:   req.EnrollmentToken,
		AgentVersion:      req.AgentVersion,
		RequestedHostname: req.Hostname,
		CsrPem:            req.CSRPEM,
		Labels:            req.Labels,
		BootstrapNonce:    req.BootstrapNonce,
	}

	resp, err := c.enroll.Enroll(ctx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("enrollment failed: %w", err)
	}

	return &GRPCEnrollResponse{
		TenantID:                 resp.TenantId,
		SiteID:                   resp.SiteId,
		HostID:                   resp.HostId,
		AgentID:                  resp.AgentId,
		ClientCertificatePEM:     resp.ClientCertificatePem,
		CACertificatePEM:         resp.CaCertificatePem,
		RefreshToken:             resp.RefreshToken,
		HeartbeatEndpoint:        resp.HeartbeatEndpoint,
		HeartbeatIntervalSeconds: int(resp.HeartbeatIntervalSeconds),
	}, nil
}

// Heartbeat sends a heartbeat to the control plane
func (c *GRPCClient) Heartbeat(ctx context.Context, req GRPCHeartbeatRequest) (*GRPCHeartbeatResponse, error) {
	// Convert microVMs to proto format
	microvms := make([]*controlplanev1.MicroVMStatus, 0, len(req.MicroVMs))
	for _, vm := range req.MicroVMs {
		state := controlplanev1.MicroVMState_MICRO_VM_STATE_UNSPECIFIED
		switch vm.State {
		case "CREATING":
			state = controlplanev1.MicroVMState_MICRO_VM_STATE_CREATING
		case "STOPPED":
			state = controlplanev1.MicroVMState_MICRO_VM_STATE_STOPPED
		case "RUNNING":
			state = controlplanev1.MicroVMState_MICRO_VM_STATE_RUNNING
		case "DELETING":
			state = controlplanev1.MicroVMState_MICRO_VM_STATE_DELETING
		case "ERROR":
			state = controlplanev1.MicroVMState_MICRO_VM_STATE_ERROR
		}
		
		microvms = append(microvms, &controlplanev1.MicroVMStatus{
			VmId:      vm.ID,
			Name:      vm.Name,
			State:     state,
			VcpuCount: uint32(vm.VCPUCount),
			MemoryMib: uint64(vm.MemoryMiB),
		})
	}

	grpcReq := &controlplanev1.HeartbeatRequest{
		TenantId:     req.TenantID,
		SiteId:       req.SiteID,
		HostId:       req.HostID,
		AgentId:      req.AgentID,
		HeartbeatSeq: req.HeartbeatSeq,
		SentAt:       nil, // Will be set by server
		AgentStatus: &controlplanev1.AgentStatus{
			Version:       req.AgentVersion,
			Os:            req.OS,
			Arch:          req.Arch,
			KernelVersion: req.KernelVersion,
		},
		HostFacts: &controlplanev1.HostFacts{
			CpuCoresTotal:            uint32(req.CPUCoresTotal),
			MemoryBytesTotal:         uint64(req.MemoryBytesTotal),
			StorageBytesTotal:        uint64(req.StorageBytesTotal),
			KvmAvailable:             req.KVMAvailable,
			CloudHypervisorAvailable: req.CloudHypervisorAvailable,
		},
		Microvms: microvms,
	}

	resp, err := c.agent.Heartbeat(ctx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("heartbeat failed: %w", err)
	}

	// Convert pending plans
	plans := make([]GRPCPlan, 0, len(resp.PendingPlans))
	for _, p := range resp.PendingPlans {
		plan := GRPCPlan{
			PlanID: p.PlanId,
		}
		
		actions := make([]GRPCPlanAction, 0, len(p.Operations))
		for _, op := range p.Operations {
			actionType := "unknown"
			switch op.Operation {
			case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_CREATE:
				actionType = "create"
			case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_START:
				actionType = "start"
			case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_STOP:
				actionType = "stop"
			case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_DELETE:
				actionType = "delete"
			}
			
			actions = append(actions, GRPCPlanAction{
				ActionID: op.OperationId,
				Type:     actionType,
				Timeout:  int(op.TimeoutSeconds),
			})
		}
		plan.Actions = actions
		plans = append(plans, plan)
	}

	return &GRPCHeartbeatResponse{
		NextHeartbeatSeconds: int(resp.NextHeartbeatSeconds),
		PendingPlans:         plans,
		RotateCertificate:    resp.RotateCertificate,
	}, nil
}

// StreamLogs opens a log streaming connection to the control plane
func (c *GRPCClient) StreamLogs(ctx context.Context) (*GRPCLogStream, error) {
	stream, err := c.agent.StreamLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open log stream: %w", err)
	}

	return &GRPCLogStream{
		stream: stream,
	}, nil
}

// GRPCLogStream wraps the gRPC log streaming client
type GRPCLogStream struct {
	stream controlplanev1.AgentControlService_StreamLogsClient
}

// Send sends a log frame to the control plane
func (s *GRPCLogStream) Send(frame *GRPCLogFrame) error {
	severity := controlplanev1.LogSeverity_LOG_SEVERITY_UNSPECIFIED
	switch frame.Severity {
	case "DEBUG":
		severity = controlplanev1.LogSeverity_LOG_SEVERITY_DEBUG
	case "INFO":
		severity = controlplanev1.LogSeverity_LOG_SEVERITY_INFO
	case "WARN":
		severity = controlplanev1.LogSeverity_LOG_SEVERITY_WARN
	case "ERROR":
		severity = controlplanev1.LogSeverity_LOG_SEVERITY_ERROR
	}

	grpcFrame := &controlplanev1.LogFrame{
		TenantId:    frame.TenantID,
		SiteId:      frame.SiteID,
		AgentId:     frame.AgentID,
		PlanId:      frame.PlanID,
		ExecutionId: frame.ExecutionID,
		OperationId: frame.OperationID,
		VmId:        frame.VMID,
		Sequence:    frame.Sequence,
		Severity:    severity,
		Message:     frame.Message,
		Eof:         frame.EOF,
	}

	return s.stream.Send(grpcFrame)
}

// CloseAndRecv closes the stream and receives the response
func (s *GRPCLogStream) CloseAndRecv() (acceptedFrames, droppedFrames uint64, err error) {
	resp, err := s.stream.CloseAndRecv()
	if err != nil {
		return 0, 0, err
	}
	return resp.AcceptedFrames, resp.DroppedFrames, nil
}

// GRPCLogFrame represents a log frame for streaming
type GRPCLogFrame struct {
	TenantID    string
	SiteID      string
	AgentID     string
	PlanID      string
	ExecutionID string
	OperationID string
	VMID        string
	Sequence    uint64
	Severity    string
	Message     string
	EOF         bool
}
