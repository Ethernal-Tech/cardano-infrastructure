package gouroboros

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// TLSConfig holds the settings for a mutual TLS connection to the node
type TLSConfig struct {
	CAFile     string `json:"caFile"`
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
	ServerName string `json:"serverName,omitempty"`
}

// ClientConfig loads the files and returns a client config that trusts only CAFile.
// ServerName falls back to defaultServerName (the host part of NodeAddress).
func (c *TLSConfig) ClientConfig(defaultServerName string) (*tls.Config, error) {
	if c.CAFile == "" {
		return nil, errors.New("tls caFile is empty")
	}

	if c.CertFile == "" {
		return nil, errors.New("tls certFile is empty")
	}

	if c.KeyFile == "" {
		return nil, errors.New("tls keyFile is empty")
	}

	caPEM, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("tls caFile: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("tls caFile %s: no PEM certificates found", c.CAFile)
	}

	certPEM, err := os.ReadFile(c.CertFile)
	if err != nil {
		return nil, fmt.Errorf("tls certFile: %w", err)
	}

	keyPEM, err := os.ReadFile(c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("tls keyFile: %w", err)
	}

	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls certFile/keyFile: %w", err)
	}

	serverName := c.ServerName
	if serverName == "" {
		serverName = defaultServerName
	}

	if serverName == "" {
		return nil, errors.New("tls serverName is empty and no default server name is available")
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{pair},
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// DialNode dials NodeAddress: plain tcp/unix when config.TLS is nil, else TLS over tcp.
func DialNode(config *BlockSyncerConfig, timeout time.Duration) (net.Conn, error) {
	protocol := config.Protocol()
	netDialer := &net.Dialer{Timeout: timeout}

	if config.TLS == nil {
		return netDialer.Dial(protocol, config.NodeAddress)
	}

	if protocol == ProtocolUnix {
		return nil, fmt.Errorf("tls is not supported with a unix socket address %s", config.NodeAddress)
	}

	fallbackName := ""

	if config.TLS.ServerName == "" {
		host, err := defaultServerName(config.NodeAddress)
		if err != nil {
			return nil, err
		}

		fallbackName = host
	}

	tlsConfig, err := config.TLS.ClientConfig(fallbackName)
	if err != nil {
		return nil, err
	}

	// the dialer timeout covers connect and TLS handshake together
	conn, err := (&tls.Dialer{NetDialer: netDialer, Config: tlsConfig}).
		DialContext(context.Background(), protocol, config.NodeAddress)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// defaultServerName returns the host part of nodeAddress (brackets of IPv6 literals removed)
func defaultServerName(nodeAddress string) (string, error) {
	host, _, err := net.SplitHostPort(nodeAddress)
	if err != nil {
		return "", fmt.Errorf("tls serverName is empty and nodeAddress %q has no host:port: %w", nodeAddress, err)
	}

	return host, nil
}
