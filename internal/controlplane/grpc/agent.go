package grpc

import (
	"context"
	"io"
	"strings"
	"time"

	controlplanev1 "github.com/kubedoio/n-kudo/api/proto/controlplane/v1"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Heartbeat handles agent heartbeat requests
func (s *Server) Heartbeat(ctx context.Context, req *controlplanev1.HeartbeatRequest) (*controlplanev1.HeartbeatResponse, error) {
	// Get agent from context (set by mTLS auth interceptor)
	agent, ok := GetAgentFromContext(ctx)
	if !ok {
		// For now, allow without mTLS for testing - in production, require mTLS
		// return nil, status.Errorf(codes.Unauthenticated, "agent authentication required")
		
		// Try to get agent from request
		if req.AgentId == "" {
			return nil, status.Errorf(codes.InvalidArgument, "agent_id is required")
		}
		
		var err error
		agent, err = s.repo.GetAgentByID(ctx, req.AgentId)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unknown agent")
		}
	}

	// Validate agent ID matches if provided in request
	if req.AgentId != "" && req.AgentId != agent.ID {
		return nil, status.Errorf(codes.PermissionDenied, "agent_id mismatch")
	}

	// Convert microVMs from proto to store format
	vms := make([]store.MicroVMHeartbeat, 0, len(req.Microvms))
	for _, vm := range req.Microvms {
		if strings.TrimSpace(vm.VmId) == "" {
			continue
		}
		state := vm.State.String()
		if state == "" {
			state = "CREATING"
		}
		vms = append(vms, store.MicroVMHeartbeat{
			ID:        vm.VmId,
			Name:      firstNonEmpty(vm.Name, vm.VmId),
			State:     state,
			VCPUCount: int(vm.VcpuCount),
			MemoryMiB: int64(vm.MemoryMib),
			UpdatedAt: time.Now().UTC(),
		})
	}

	// Convert execution updates
	execUpdates := make([]store.ExecutionUpdate, 0, len(req.ExecutionUpdates))
	for _, update := range req.ExecutionUpdates {
		execUpdates = append(execUpdates, store.ExecutionUpdate{
			ExecutionID:  update.ExecutionId,
			State:        update.State.String(),
			ErrorCode:    update.ErrorCode,
			ErrorMessage: update.ErrorMessage,
			UpdatedAt:    time.Now().UTC(),
		})
	}

	// Extract host facts
	hostname := "unknown"
	var cpuCores int
	var memoryTotal int64
	var storageTotal int64
	var kvmAvailable bool
	var chvAvailable bool

	if req.HostFacts != nil {
		cpuCores = int(req.HostFacts.CpuCoresTotal)
		memoryTotal = int64(req.HostFacts.MemoryBytesTotal)
		storageTotal = int64(req.HostFacts.StorageBytesTotal)
		kvmAvailable = req.HostFacts.KvmAvailable
		chvAvailable = req.HostFacts.CloudHypervisorAvailable
	}

	if req.AgentStatus != nil {
		hostname = valueOr(hostname, "unknown")
	}

	// Ingest heartbeat
	err := s.repo.IngestHeartbeat(ctx, store.Heartbeat{
		AgentID:                  agent.ID,
		HeartbeatSeq:             req.HeartbeatSeq,
		AgentVersion:             valueOr(req.AgentStatus.GetVersion(), agent.AgentVersion),
		OS:                       valueOr(req.AgentStatus.GetOs(), agent.OS),
		Arch:                     valueOr(req.AgentStatus.GetArch(), agent.Arch),
		KernelVersion:            valueOr(req.AgentStatus.GetKernelVersion(), agent.KernelVersion),
		Hostname:                 hostname,
		CPUCoresTotal:            cpuCores,
		MemoryBytesTotal:         memoryTotal,
		StorageBytesTotal:        storageTotal,
		KVMAvailable:             kvmAvailable,
		CloudHypervisorAvailable: chvAvailable,
		MicroVMs:                 vms,
		ExecutionUpdates:         execUpdates,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to ingest heartbeat: %v", err)
	}

	// Lease pending plans
	maxPlans := 2
	if s.maxPlansPerHeartbeat > 0 {
		maxPlans = s.maxPlansPerHeartbeat
	}
	
	leaseTTL := 45 * time.Second
	if s.planLeaseTTL > 0 {
		leaseTTL = s.planLeaseTTL
	}

	pending, err := s.repo.LeasePendingPlans(ctx, agent.ID, maxPlans, leaseTTL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to lease plans: %v", err)
	}

	// Convert plans to proto format
	protoPlans := make([]*controlplanev1.Plan, 0, len(pending))
	for _, plan := range pending {
		protoPlan := &controlplanev1.Plan{
			PlanId:        plan.PlanID,
			PlanVersion:   1, // Default version
			SiteId:        agent.SiteID,
			IdempotencyKey: "",
			CreatedAt:     nil,
		}
		
		// Convert actions to operations
		operations := make([]*controlplanev1.PlanOperation, 0, len(plan.Actions))
		for _, action := range plan.Actions {
			op := &controlplanev1.PlanOperation{
				OperationId: action.OperationID,
				VmId:        action.VMID,
			}
			
			// Map operation type
			switch action.OperationType {
			case "CREATE":
				op.Operation = controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_CREATE
			case "START":
				op.Operation = controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_START
			case "STOP":
				op.Operation = controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_STOP
			case "DELETE":
				op.Operation = controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_DELETE
			default:
				op.Operation = controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_UNSPECIFIED
			}
			
			operations = append(operations, op)
		}
		protoPlan.Operations = operations
		protoPlans = append(protoPlans, protoPlan)
	}

	// Calculate next heartbeat interval
	nextHeartbeatSeconds := int32(15)
	if s.heartbeatInterval > 0 {
		nextHeartbeatSeconds = int32(s.heartbeatInterval.Seconds())
	}

	return &controlplanev1.HeartbeatResponse{
		NextHeartbeatSeconds: nextHeartbeatSeconds,
		PendingPlans:         protoPlans,
		RotateCertificate:    false, // TODO: implement certificate rotation logic
	}, nil
}

// StreamLogs handles bidirectional log streaming from agents
func (s *Server) StreamLogs(stream controlplanev1.AgentControlService_StreamLogsServer) error {
	// Get agent from context
	agent, ok := GetAgentFromContext(stream.Context())
	if !ok {
		// For testing, allow without mTLS
		// In production, require mTLS auth
	}

	var acceptedFrames, droppedFrames uint64
	var entries []store.LogIngestEntry
	var agentID string

	if ok {
		agentID = agent.ID
	}

	// Process incoming log frames
	for {
		frame, err := stream.Recv()
		if err == io.EOF {
			// End of stream, ingest collected logs
			if len(entries) > 0 && agentID != "" {
				_, _, err := s.repo.IngestLogs(stream.Context(), store.LogIngest{
					AgentID: agentID,
					Entries: entries,
				})
				if err != nil {
					return status.Errorf(codes.Internal, "failed to ingest logs: %v", err)
				}
			}
			
			return stream.SendAndClose(&controlplanev1.StreamLogsResponse{
				AcceptedFrames: acceptedFrames,
				DroppedFrames:  droppedFrames,
			})
		}
		if err != nil {
			return err
		}

		// Update agent ID from frame if not set
		if agentID == "" && frame.AgentId != "" {
			agentID = frame.AgentId
		}

		// Validate frame
		if frame.ExecutionId == "" {
			droppedFrames++
			continue
		}

		// Convert severity
		severity := frame.Severity.String()

		// Handle timestamp
		emittedAt := time.Now().UTC()
		if frame.EmittedAt != nil {
			emittedAt = frame.EmittedAt.AsTime()
		}

		entries = append(entries, store.LogIngestEntry{
			ExecutionID: frame.ExecutionId,
			Sequence:    int64(frame.Sequence),
			Severity:    severity,
			Message:     frame.Message,
			EmittedAt:   emittedAt,
		})

		acceptedFrames++

		// Batch ingest if we have enough entries
		if len(entries) >= 100 {
			if agentID != "" {
				_, dropped, err := s.repo.IngestLogs(stream.Context(), store.LogIngest{
					AgentID: agentID,
					Entries: entries,
				})
				if err != nil {
					return status.Errorf(codes.Internal, "failed to ingest logs: %v", err)
				}
				droppedFrames += uint64(dropped)
			}
			entries = entries[:0] // Clear slice but keep capacity
		}
	}
}
