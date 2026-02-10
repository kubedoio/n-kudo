package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	controlplanev1 "github.com/kubedoio/n-kudo/api/proto/controlplane/v1"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// CAInterface defines the interface needed from the CA
type CAInterface interface {
	CertPEM() []byte
	CertPool() *x509.CertPool
	SignAgentCSR(csrPEM []byte, agentID, tenantID, siteID string, ttl time.Duration) (certPEM []byte, serial string, err error)
}

// Server implements the gRPC control plane server
type Server struct {
	controlplanev1.UnimplementedEnrollmentServiceServer
	controlplanev1.UnimplementedAgentControlServiceServer
	controlplanev1.UnimplementedTenantControlServiceServer

	cfg                   Config
	repo                  store.Repo
	ca                    CAInterface
	heartbeatInterval     time.Duration
	planLeaseTTL          time.Duration
	maxPlansPerHeartbeat  int
	agentCertTTL          time.Duration

	grpcServer *grpc.Server
	listener   net.Listener
	mu         sync.RWMutex
	started    bool
}

// NewServer creates a new gRPC server instance
func NewServer(cfg Config, repo store.Repo, ca CAInterface, heartbeatInterval time.Duration, planLeaseTTL time.Duration, maxPlansPerHeartbeat int, agentCertTTL time.Duration) *Server {
	return &Server{
		cfg:                  cfg,
		repo:                 repo,
		ca:                   ca,
		heartbeatInterval:    heartbeatInterval,
		planLeaseTTL:         planLeaseTTL,
		maxPlansPerHeartbeat: maxPlansPerHeartbeat,
		agentCertTTL:         agentCertTTL,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	// Create listener
	listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Build server options
	opts := s.buildServerOptions()

	// Create gRPC server
	s.grpcServer = grpc.NewServer(opts...)

	// Register services
	controlplanev1.RegisterEnrollmentServiceServer(s.grpcServer, s)
	controlplanev1.RegisterAgentControlServiceServer(s.grpcServer, s)
	controlplanev1.RegisterTenantControlServiceServer(s.grpcServer, s)

	s.started = true

	log.Printf("[grpc] Server starting on %s", s.cfg.ListenAddr)

	// Start serving in a goroutine
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			log.Printf("[grpc] Server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	log.Println("[grpc] Stopping server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("[grpc] Server stopped gracefully")
	case <-ctx.Done():
		log.Println("[grpc] Graceful stop timeout, forcing shutdown")
		s.grpcServer.Stop()
	}

	s.started = false
	return nil
}

// buildServerOptions builds gRPC server options including TLS and interceptors
func (s *Server) buildServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		// Keepalive settings
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		// Unary interceptors
		grpc.ChainUnaryInterceptor(
			LoggingInterceptor(),
			RecoveryInterceptor(),
		),
		// Stream interceptors
		grpc.ChainStreamInterceptor(
			StreamLoggingInterceptor(),
			StreamRecoveryInterceptor(),
		),
	}

	// Configure TLS if CA is available
	if s.ca != nil {
		tlsConfig := s.buildTLSConfig()
		if tlsConfig != nil {
			opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		}
	}

	return opts
}

// buildTLSConfig builds TLS configuration for mTLS support
func (s *Server) buildTLSConfig() *tls.Config {
	if s.ca == nil {
		return nil
	}

	// Get server certificate from CA or use configured cert files
	var cert tls.Certificate
	var err error

	if s.cfg.TLSConfig.CertFile != "" && s.cfg.TLSConfig.KeyFile != "" {
		cert, err = tls.LoadX509KeyPair(s.cfg.TLSConfig.CertFile, s.cfg.TLSConfig.KeyFile)
		if err != nil {
			log.Printf("[grpc] Failed to load TLS cert/key: %v", err)
			return nil
		}
	} else {
		// Generate a self-signed cert using the CA
		cert, err = s.generateServerCert()
		if err != nil {
			log.Printf("[grpc] Failed to generate server cert: %v", err)
			return nil
		}
	}

	// Build CA pool
	pool := s.ca.CertPool()

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS13,
	}
}

// generateServerCert generates a server certificate using the internal CA
func (s *Server) generateServerCert() (tls.Certificate, error) {
	// This is a simplified version - in production you'd want to properly
	// generate a server certificate signed by the CA
	// For now, return an error to indicate that cert files should be provided
	return tls.Certificate{}, fmt.Errorf("TLS cert files must be provided")
}

// Addr returns the server address
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// IsStarted returns whether the server is started
func (s *Server) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}

// Helper functions for validation
func validateRequiredString(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return status.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}
	return nil
}

// RecoveryInterceptor recovers from panics in handlers
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[grpc] Panic recovered in %s: %v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecoveryInterceptor recovers from panics in stream handlers
func StreamRecoveryInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[grpc] Panic recovered in stream %s: %v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(srv, stream)
	}
}
