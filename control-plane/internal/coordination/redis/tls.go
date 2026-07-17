package redis

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"
)

func newTLSConfig(cfg Config) (*tls.Config, error) {
	if !cfg.TLS {
		return nil, nil
	}
	caFile := strings.TrimSpace(cfg.TLSCAFile)
	if caFile == "" {
		return nil, fmt.Errorf("redis TLS CA file is required when TLS is enabled")
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read redis TLS CA file: %w", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("redis TLS CA file contains no certificate")
	}
	serverName := strings.TrimSpace(cfg.TLSServerName)
	if serverName == "" {
		serverName, _, err = net.SplitHostPort(strings.TrimSpace(cfg.Addr))
		if err != nil || serverName == "" {
			return nil, fmt.Errorf("redis TLS server name is required for address %q", cfg.Addr)
		}
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: serverName}, nil
}
