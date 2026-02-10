package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	controlplanev1 "github.com/kubedoio/n-kudo/api/proto/controlplane/v1"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Enroll handles agent enrollment requests
func (s *Server) Enroll(ctx context.Context, req *controlplanev1.EnrollRequest) (*controlplanev1.EnrollResponse, error) {
	// Validate required fields
	hostname := firstNonEmpty(req.RequestedHostname, "")
	if req.EnrollmentToken == "" || req.CsrPem == "" || hostname == "" {
		return nil, status.Errorf(codes.InvalidArgument, "enrollment_token, requested_hostname and csr_pem are required")
	}

	// Consume enrollment token
	consume, err := s.repo.ConsumeEnrollmentToken(ctx, hashString(req.EnrollmentToken), time.Now().UTC())
	if err != nil {
		if err == store.ErrTokenInvalid {
			return nil, status.Errorf(codes.Unauthenticated, "invalid or expired enrollment token")
		}
		return nil, status.Errorf(codes.Internal, "failed to validate enrollment token")
	}

	// Generate agent ID
	agentID := uuid.NewString()

	// Sign the CSR
	certPEM, certSerial, err := s.ca.SignAgentCSR([]byte(req.CsrPem), agentID, consume.TenantID, consume.SiteID, s.agentCertTTL)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid csr: %v", err)
	}

	// Generate refresh token
	refreshToken, err := randomToken(32)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token")
	}

	// Create agent from enrollment
	agent, err := s.repo.CreateAgentFromEnrollment(ctx, consume.TokenID, store.Agent{
		ID:               agentID,
		TenantID:         consume.TenantID,
		SiteID:           consume.SiteID,
		HostID:           uuid.NewString(),
		CertSerial:       certSerial,
		RefreshTokenHash: hashString(refreshToken),
		AgentVersion:     valueOr(req.AgentVersion, "mvp1"),
		OS:               "linux",  // Extract from host facts if available
		Arch:             "amd64",  // Extract from host facts if available
		KernelVersion:    "",       // Extract from host facts if available
	}, hostname)

	if err != nil {
		if err == store.ErrConflict {
			return nil, status.Errorf(codes.AlreadyExists, "agent already exists for host")
		}
		return nil, status.Errorf(codes.Internal, "failed to create agent")
	}

	// Calculate heartbeat interval
	heartbeatInterval := int32(15) // default 15 seconds
	if s.heartbeatInterval > 0 {
		heartbeatInterval = int32(s.heartbeatInterval.Seconds())
	}

	return &controlplanev1.EnrollResponse{
		TenantId:                 agent.TenantID,
		SiteId:                   agent.SiteID,
		HostId:                   agent.HostID,
		AgentId:                  agent.ID,
		ClientCertificatePem:     string(certPEM),
		CaCertificatePem:         string(s.ca.CertPEM()),
		RefreshToken:             refreshToken,
		HeartbeatEndpoint:        "/v1/heartbeat",
		HeartbeatIntervalSeconds: heartbeatInterval,
	}, nil
}

// Helper functions

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	// This should use the same implementation as in api/server.go
	// For now, generate a simple UUID-based token
	return uuid.NewString() + uuid.NewString(), nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
