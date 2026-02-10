package grpc

// Config holds the gRPC server configuration
type Config struct {
	Enabled    bool
	ListenAddr string
	TLSConfig  TLSConfig
}

// TLSConfig holds TLS configuration for the gRPC server
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// DefaultConfig returns a default configuration for the gRPC server
func DefaultConfig() Config {
	return Config{
		Enabled:    false,
		ListenAddr: ":50051",
		TLSConfig: TLSConfig{
			CertFile: "",
			KeyFile:  "",
			CAFile:   "",
		},
	}
}
