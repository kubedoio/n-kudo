package grpc

import (
	"context"
	"log"
	"strings"
	"time"

	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// context keys for storing auth information
type contextKey string

const (
	ctxTenantIDKey contextKey = "tenant_id"
	ctxAgentIDKey  contextKey = "agent_id"
	ctxAPIKeyKey   contextKey = "api_key"
)

// LoggingInterceptor logs unary RPC calls
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract client info
		clientIP := extractClientIP(ctx)

		// Call handler
		resp, err := handler(ctx, req)

		// Log the call
		duration := time.Since(start)
		statusCode := codes.OK
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		}

		log.Printf("[grpc] %s | %s | %v | %s | %s",
			clientIP,
			info.FullMethod,
			duration,
			statusCode.String(),
			errorMessage(err),
		)

		return resp, err
	}
}

// StreamLoggingInterceptor logs stream RPC calls
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Extract client info
		ctx := stream.Context()
		clientIP := extractClientIP(ctx)

		// Call handler
		err := handler(srv, stream)

		// Log the call
		duration := time.Since(start)
		statusCode := codes.OK
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		}

		log.Printf("[grpc] %s | %s | stream | %v | %s | %s",
			clientIP,
			info.FullMethod,
			duration,
			statusCode.String(),
			errorMessage(err),
		)

		return err
	}
}

// APIKeyAuthInterceptor validates API keys for tenant-scoped endpoints
func APIKeyAuthInterceptor(repo store.Repo) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for enrollment endpoints
		if isPublicEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Skip auth for agent endpoints (they use mTLS)
		if isAgentEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract API key from metadata
		apiKey, err := extractAPIKey(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "missing or invalid API key: %v", err)
		}

		// Validate API key against database
		validation, err := repo.ValidateAPIKey(ctx, hashString(apiKey))
		if err != nil {
			if err == store.ErrUnauthorized {
				return nil, status.Errorf(codes.Unauthenticated, "invalid API key")
			}
			return nil, status.Errorf(codes.Internal, "failed to validate API key: %v", err)
		}

		// Add tenant ID to context
		ctx = context.WithValue(ctx, ctxTenantIDKey, validation.TenantID)

		return handler(ctx, req)
	}
}

// MTLSAuthInterceptor validates mTLS client certificates for agent endpoints
func MTLSAuthInterceptor(repo store.Repo) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Only apply to agent endpoints
		if !isAgentEndpoint(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract agent ID from client certificate
		agentID, err := extractAgentIDFromCert(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid client certificate: %v", err)
		}

		// Verify agent exists
		agent, err := repo.GetAgentByID(ctx, agentID)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unknown agent")
		}

		// Add agent info to context
		ctx = context.WithValue(ctx, ctxAgentIDKey, agent)

		return handler(ctx, req)
	}
}

// Helper functions

func extractClientIP(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}

func extractAPIKey(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	// Check X-API-Key header
	keys := md.Get("x-api-key")
	if len(keys) > 0 && strings.TrimSpace(keys[0]) != "" {
		return strings.TrimSpace(keys[0]), nil
	}

	// Also check authorization header with Bearer prefix
	authHeaders := md.Get("authorization")
	if len(authHeaders) > 0 {
		auth := strings.TrimSpace(authHeaders[0])
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return strings.TrimSpace(auth[7:]), nil
		}
	}

	return "", status.Errorf(codes.Unauthenticated, "API key not found")
}

func extractAgentIDFromCert(ctx context.Context) (string, error) {
	// This would extract the agent ID from the mTLS client certificate
	// The actual implementation depends on how the certificates are structured
	// For now, we'll extract it from the context if it was set by TLS credentials
	// In a real implementation, you'd parse the peer certificate from the context
	
	// Placeholder - in production, parse the peer.AuthInfo
	return "", status.Errorf(codes.Unauthenticated, "mTLS not configured")
}

func isPublicEndpoint(method string) bool {
	publicMethods := []string{
		"/nkudo.controlplane.v1.EnrollmentService/Enroll",
		"/nkudo.controlplane.v1.AgentControlService/Heartbeat",
		"/grpc.health.v1.Health/Check",
	}
	for _, m := range publicMethods {
		if method == m {
			return true
		}
	}
	return false
}

func isAgentEndpoint(method string) bool {
	agentMethods := []string{
		"/nkudo.controlplane.v1.AgentControlService/Heartbeat",
		"/nkudo.controlplane.v1.AgentControlService/StreamLogs",
	}
	for _, m := range agentMethods {
		if method == m {
			return true
		}
	}
	return false
}

func errorMessage(err error) string {
	if err == nil {
		return "-"
	}
	msg := err.Error()
	if len(msg) > 100 {
		return msg[:100] + "..."
	}
	return msg
}

// hashString creates a SHA256 hash of a string (placeholder)
// This is defined in enrollment.go - using that implementation
func hashStringPlaceholder(s string) string {
	return s
}

// GetTenantIDFromContext extracts tenant ID from context
func GetTenantIDFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(ctxTenantIDKey).(string)
	return tenantID, ok
}

// GetAgentFromContext extracts agent from context
func GetAgentFromContext(ctx context.Context) (store.Agent, bool) {
	agent, ok := ctx.Value(ctxAgentIDKey).(store.Agent)
	return agent, ok
}
