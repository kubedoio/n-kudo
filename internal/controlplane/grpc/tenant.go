package grpc

import (
	"context"
	"strings"

	controlplanev1 "github.com/kubedoio/n-kudo/api/proto/controlplane/v1"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ApplyPlan handles plan application requests from tenants
func (s *Server) ApplyPlan(ctx context.Context, req *controlplanev1.ApplyPlanRequest) (*controlplanev1.ApplyPlanResponse, error) {
	// Get tenant ID from context (set by API key auth interceptor)
	tenantID, ok := GetTenantIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "tenant authentication required")
	}

	// Validate request
	if strings.TrimSpace(req.SiteId) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "site_id is required")
	}

	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "idempotency_key is required")
	}

	if len(req.Operations) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "operations are required")
	}

	// Verify site belongs to tenant
	belongs, err := s.repo.SiteBelongsToTenant(ctx, req.SiteId, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "site lookup failed")
	}
	if !belongs {
		return nil, status.Errorf(codes.NotFound, "site not found")
	}

	// Convert operations to store format
	actions := make([]store.ApplyPlanAction, 0, len(req.Operations))
	for _, op := range req.Operations {
		action := store.ApplyPlanAction{
			OperationID: op.OperationId,
			VMID:        op.VmId,
		}

		// Map operation type
		switch op.Operation {
		case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_CREATE:
			action.Operation = "CREATE"
			// Extract creation config if provided
			if op.CreateConfig != nil {
				action.Name = op.CreateConfig.Name
				action.VCPUCount = int(op.CreateConfig.VcpuCount)
				action.MemoryMiB = int64(op.CreateConfig.MemoryMib)
			}
		case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_START:
			action.Operation = "START"
		case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_STOP:
			action.Operation = "STOP"
		case controlplanev1.MicroVMOperation_MICRO_VM_OPERATION_DELETE:
			action.Operation = "DELETE"
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid operation type: %v", op.Operation)
		}

		actions = append(actions, action)
	}

	// Apply the plan
	result, err := s.repo.ApplyPlan(ctx, store.ApplyPlanInput{
		TenantID:        tenantID,
		SiteID:          req.SiteId,
		IdempotencyKey:  req.IdempotencyKey,
		ClientRequestID: req.ClientRequestId,
		Actions:         actions,
	})

	if err != nil {
		if err == store.ErrUnauthorized {
			return nil, status.Errorf(codes.PermissionDenied, "tenant mismatch")
		}
		return nil, status.Errorf(codes.Internal, "failed to apply plan: %v", err)
	}

	// Map plan status
	planStatus := controlplanev1.PlanStatus_PLAN_STATUS_UNSPECIFIED
	switch result.Plan.Status {
	case "PENDING":
		planStatus = controlplanev1.PlanStatus_PLAN_STATUS_PENDING
	case "IN_PROGRESS":
		planStatus = controlplanev1.PlanStatus_PLAN_STATUS_IN_PROGRESS
	case "SUCCEEDED":
		planStatus = controlplanev1.PlanStatus_PLAN_STATUS_SUCCEEDED
	case "FAILED":
		planStatus = controlplanev1.PlanStatus_PLAN_STATUS_FAILED
	case "CANCELLED":
		planStatus = controlplanev1.PlanStatus_PLAN_STATUS_CANCELLED
	}

	return &controlplanev1.ApplyPlanResponse{
		PlanId:       result.Plan.ID,
		PlanVersion:  uint64(result.Plan.PlanVersion),
		PlanStatus:   planStatus,
		Deduplicated: result.Deduplicated,
	}, nil
}

// GetStatus handles status queries for sites
func (s *Server) GetStatus(ctx context.Context, req *controlplanev1.StatusQueryRequest) (*controlplanev1.StatusQueryResponse, error) {
	// Get tenant ID from context
	tenantID, ok := GetTenantIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "tenant authentication required")
	}

	// Validate request
	if strings.TrimSpace(req.SiteId) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "site_id is required")
	}

	// Verify site belongs to tenant
	belongs, err := s.repo.SiteBelongsToTenant(ctx, req.SiteId, tenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "site lookup failed")
	}
	if !belongs {
		return nil, status.Errorf(codes.NotFound, "site not found")
	}

	// Get hosts
	hosts, err := s.repo.ListHosts(ctx, tenantID, req.SiteId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list hosts: %v", err)
	}

	// Get VMs
	vms, err := s.repo.ListVMs(ctx, tenantID, req.SiteId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list VMs: %v", err)
	}

	// Build response
	resp := &controlplanev1.StatusQueryResponse{
		Site: &controlplanev1.SiteStatus{
			SiteId:            req.SiteId,
			ConnectivityState: controlplanev1.SiteConnectivityState_SITE_CONNECTIVITY_STATE_UNSPECIFIED,
		},
		Hosts:    make([]*controlplanev1.HostStatus, 0, len(hosts)),
		Microvms: make([]*controlplanev1.MicroVMStatus, 0, len(vms)),
	}

	// Convert hosts to proto format
	for _, host := range hosts {
		// Map agent state
		agentState := controlplanev1.AgentLifecycleState_AGENT_LIFECYCLE_STATE_UNSPECIFIED
		switch host.AgentState {
		case "ONLINE":
			agentState = controlplanev1.AgentLifecycleState_AGENT_LIFECYCLE_STATE_ONLINE
		case "DEGRADED":
			agentState = controlplanev1.AgentLifecycleState_AGENT_LIFECYCLE_STATE_DEGRADED
		case "OFFLINE":
			agentState = controlplanev1.AgentLifecycleState_AGENT_LIFECYCLE_STATE_OFFLINE
		}

		hostStatus := &controlplanev1.HostStatus{
			HostId:      host.ID,
			Hostname:    host.Hostname,
			AgentState:  agentState,
			Capacity: &controlplanev1.HostFacts{
				CpuCoresTotal:            uint32(host.CPUCoresTotal),
				MemoryBytesTotal:         uint64(host.MemoryBytesTotal),
				StorageBytesTotal:        uint64(host.StorageBytesTotal),
				KvmAvailable:             host.KVMAvailable,
				CloudHypervisorAvailable: host.CloudHypervisorAvailable,
			},
		}

		if host.AgentLastHeartbeatAt != nil {
			hostStatus.LastHeartbeatAt = timestamppb.New(*host.AgentLastHeartbeatAt)
		}

		resp.Hosts = append(resp.Hosts, hostStatus)
	}

	// Convert VMs to proto format
	for _, vm := range vms {
		// Map VM state
		vmState := controlplanev1.MicroVMState_MICRO_VM_STATE_UNSPECIFIED
		switch vm.State {
		case "CREATING":
			vmState = controlplanev1.MicroVMState_MICRO_VM_STATE_CREATING
		case "STOPPED":
			vmState = controlplanev1.MicroVMState_MICRO_VM_STATE_STOPPED
		case "RUNNING":
			vmState = controlplanev1.MicroVMState_MICRO_VM_STATE_RUNNING
		case "DELETING":
			vmState = controlplanev1.MicroVMState_MICRO_VM_STATE_DELETING
		case "ERROR":
			vmState = controlplanev1.MicroVMState_MICRO_VM_STATE_ERROR
		}

		vmStatus := &controlplanev1.MicroVMStatus{
			VmId:      vm.ID,
			SiteId:    vm.SiteID,
			HostId:    vm.HostID,
			Name:      vm.Name,
			State:     vmState,
			VcpuCount: uint32(vm.VCPUCount),
			MemoryMib: uint64(vm.MemoryMiB),
		}

		if vm.LastTransitionAt != nil {
			vmStatus.UpdatedAt = timestamppb.New(*vm.LastTransitionAt)
		} else {
			vmStatus.UpdatedAt = timestamppb.New(vm.UpdatedAt)
		}

		resp.Microvms = append(resp.Microvms, vmStatus)
	}

	// TODO: Add executions and recent logs if requested

	return resp, nil
}
