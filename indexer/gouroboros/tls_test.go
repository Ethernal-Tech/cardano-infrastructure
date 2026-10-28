package gouroboros

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

const testServerName = "node.test"

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pem  []byte
}

type testPair struct {
	certPEM []byte
	keyPEM  []byte
	cert    tls.Certificate
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().UTC().Add(-time.Hour),
		NotAfter:              time.Now().UTC().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &testCA{
		cert: cert,
		key:  key,
		pem:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

func (ca *testCA) issue(t *testing.T, cn string, server bool) testPair {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UTC().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().UTC().Add(-time.Hour),
		NotAfter:     time.Now().UTC().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.DNSNames = []string{testServerName}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	return testPair{certPEM: certPEM, keyPEM: keyPEM, cert: cert}
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	return path
}

// clientTLS writes the CA and a client pair to dir and returns the matching TLSConfig
func clientTLS(t *testing.T, dir string, ca *testCA, client testPair) *TLSConfig {
	t.Helper()

	return &TLSConfig{
		CAFile:   writeFile(t, dir, "ca.pem", ca.pem),
		CertFile: writeFile(t, dir, "client.pem", client.certPEM),
		KeyFile:  writeFile(t, dir, "client.key", client.keyPEM),
	}
}

func serverTLSConfig(ca *testCA, server testPair, clientAuth tls.ClientAuthType) *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(ca.cert)

	return &tls.Config{
		Certificates: []tls.Certificate{server.cert},
		ClientCAs:    pool,
		ClientAuth:   clientAuth,
		MinVersion:   tls.VersionTLS12,
	}
}

func listenTLS(t *testing.T, cfg *tls.Config) net.Listener {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	require.NoError(t, err)

	t.Cleanup(func() { _ = ln.Close() })

	return ln
}

func listenTCP(t *testing.T) net.Listener {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = ln.Close() })

	return ln
}

func listenUnix(t *testing.T, name string) *net.UnixListener {
	t.Helper()

	ln, err := net.Listen("unix", filepath.Join(t.TempDir(), name))
	require.NoError(t, err)

	t.Cleanup(func() { _ = ln.Close() })

	return ln.(*net.UnixListener)
}

func portOf(t *testing.T, ln net.Listener) string {
	t.Helper()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	return port
}

func requireNoAccept(t *testing.T, ln net.Listener) {
	t.Helper()

	type deadliner interface{ SetDeadline(time.Time) error }

	require.NoError(t, ln.(deadliner).SetDeadline(time.Now().UTC().Add(200*time.Millisecond)))

	conn, err := ln.Accept()
	if err == nil {
		_ = conn.Close()
	}

	require.Error(t, err)

	var nerr net.Error

	require.ErrorAs(t, err, &nerr)
	require.True(t, nerr.Timeout())
}

type handshakeResult struct {
	state tls.ConnectionState
	err   error
}

// serveHandshake accepts one connection, completes the handshake and reports the result.
// The connection is returned through connCh so that the test can control its lifetime.
func serveHandshake(ln net.Listener) (<-chan handshakeResult, <-chan *tls.Conn) {
	resCh := make(chan handshakeResult, 1)
	connCh := make(chan *tls.Conn, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			resCh <- handshakeResult{err: err}

			return
		}

		tlsConn := conn.(*tls.Conn)
		err = tlsConn.Handshake()
		resCh <- handshakeResult{state: tlsConn.ConnectionState(), err: err}
		connCh <- tlsConn
	}()

	return resCh, connCh
}

func TestDialNode_MutualTLS(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	ln := listenTLS(t, serverTLSConfig(ca, ca.issue(t, "server", true), tls.RequireAndVerifyClientCert))
	resCh, connCh := serveHandshake(ln)

	cfg := &BlockSyncerConfig{
		NodeAddress: "127.0.0.1:" + portOf(t, ln),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client-1", false)),
	}
	cfg.TLS.ServerName = testServerName

	conn, err := DialNode(cfg, 2*time.Second)
	require.NoError(t, err)

	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	require.True(t, ok)
	require.True(t, tlsConn.ConnectionState().HandshakeComplete)
	require.Equal(t, uint16(tls.VersionTLS13), tlsConn.ConnectionState().Version)

	res := <-resCh
	require.NoError(t, res.err)
	require.Len(t, res.state.PeerCertificates, 1)
	require.Equal(t, "client-1", res.state.PeerCertificates[0].Subject.CommonName)

	(<-connCh).Close()
}

func TestDialNode_PlainTCP(t *testing.T) {
	t.Parallel()

	ln := listenTCP(t)
	go func() {
		if c, err := ln.Accept(); err == nil {
			defer c.Close()

			_, _ = io.Copy(io.Discard, c)
		}
	}()

	conn, err := DialNode(&BlockSyncerConfig{NodeAddress: ln.Addr().String()}, time.Second)
	require.NoError(t, err)

	defer conn.Close()

	_, ok := conn.(*net.TCPConn)
	require.True(t, ok)
}

func TestDialNode_PlainUnix(t *testing.T) {
	t.Parallel()

	ln := listenUnix(t, "p.sock")
	go func() {
		if c, err := ln.Accept(); err == nil {
			defer c.Close()

			_, _ = io.Copy(io.Discard, c)
		}
	}()

	conn, err := DialNode(&BlockSyncerConfig{NodeAddress: ln.Addr().String()}, time.Second)
	require.NoError(t, err)

	defer conn.Close()

	_, ok := conn.(*net.UnixConn)
	require.True(t, ok)
}

func TestDialNode_WrongCA(t *testing.T) {
	t.Parallel()

	ca, otherCA := newTestCA(t), newTestCA(t)
	ln := listenTLS(t, serverTLSConfig(otherCA, otherCA.issue(t, "server", true), tls.NoClientCert))

	go func() {
		if c, err := ln.Accept(); err == nil {
			_ = c.(*tls.Conn).Handshake()
			_ = c.Close()
		}
	}()

	cfg := &BlockSyncerConfig{
		NodeAddress: "127.0.0.1:" + portOf(t, ln),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	}

	_, err := DialNode(cfg, 2*time.Second)
	require.Error(t, err)

	var uae x509.UnknownAuthorityError

	require.ErrorAs(t, err, &uae)
}

func TestDialNode_HandshakeTimeout(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	ln := listenTCP(t)

	var held []net.Conn

	go func() {
		if c, err := ln.Accept(); err == nil {
			held = append(held, c) // accept, never answer
		}
	}()

	t.Cleanup(func() {
		for _, c := range held {
			_ = c.Close()
		}
	})

	cfg := &BlockSyncerConfig{
		NodeAddress: "127.0.0.1:" + portOf(t, ln),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	}

	start := time.Now().UTC()
	_, err := DialNode(cfg, 200*time.Millisecond)

	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second)

	var nerr net.Error

	require.ErrorAs(t, err, &nerr)
	require.True(t, nerr.Timeout())
}

func TestDialNode_WrongServerName(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	ln := listenTLS(t, serverTLSConfig(ca, ca.issue(t, "server", true), tls.NoClientCert))

	go func() {
		if c, err := ln.Accept(); err == nil {
			_ = c.(*tls.Conn).Handshake()
			_ = c.Close()
		}
	}()

	cfg := &BlockSyncerConfig{
		NodeAddress: "127.0.0.1:" + portOf(t, ln),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	}
	cfg.TLS.ServerName = "other.test"

	_, err := DialNode(cfg, 2*time.Second)
	require.Error(t, err)

	var herr x509.HostnameError

	require.ErrorAs(t, err, &herr)
}

func TestDialNode_RejectsTLS12(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	srvCfg := serverTLSConfig(ca, ca.issue(t, "server", true), tls.NoClientCert)
	srvCfg.MaxVersion = tls.VersionTLS12

	ln := listenTLS(t, srvCfg)
	go func() {
		if c, err := ln.Accept(); err == nil {
			_ = c.(*tls.Conn).Handshake()
			_ = c.Close()
		}
	}()

	cfg := &BlockSyncerConfig{
		NodeAddress: "127.0.0.1:" + portOf(t, ln),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	}

	_, err := DialNode(cfg, 2*time.Second)
	require.Error(t, err)
	require.ErrorContains(t, err, "protocol version")
}

func TestDialNode_TLSWithUnixSocket(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	ln := listenUnix(t, "n.sock")

	cfg := &BlockSyncerConfig{
		NodeAddress: ln.Addr().String(),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	}

	_, err := DialNode(cfg, time.Second)
	require.Error(t, err)
	require.ErrorContains(t, err, "unix")
	require.ErrorContains(t, err, "TLS")

	requireNoAccept(t, ln)
}

func TestDialNode_EmptyTLSConfigDoesNotDialPlain(t *testing.T) {
	t.Parallel()

	ln := listenTCP(t)

	_, err := DialNode(&BlockSyncerConfig{NodeAddress: ln.Addr().String(), TLS: &TLSConfig{}}, time.Second)
	require.Error(t, err)
	require.ErrorContains(t, err, "tls caFile")

	requireNoAccept(t, ln)
}

func TestDialNode_ServerNameFallsBackToHost(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	ln := listenTLS(t, serverTLSConfig(ca, ca.issue(t, "server", true), tls.RequireAndVerifyClientCert))
	resCh, connCh := serveHandshake(ln)

	cfg := &BlockSyncerConfig{
		NodeAddress: "127.0.0.1:" + portOf(t, ln),
		TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	}

	conn, err := DialNode(cfg, 2*time.Second)
	require.NoError(t, err)

	defer conn.Close()

	res := <-resCh
	require.NoError(t, res.err)
	// IP literals are not sent as SNI, so the proof is that verification against the IP SAN passed
	require.True(t, conn.(*tls.Conn).ConnectionState().HandshakeComplete)

	(<-connCh).Close()
}

func TestDialNode_NodeAddressHostParsing(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)

	t.Run("no port", func(t *testing.T) {
		t.Parallel()

		cfg := &BlockSyncerConfig{
			NodeAddress: testServerName,
			TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
		}

		_, err := DialNode(cfg, time.Second)
		require.Error(t, err)
		require.ErrorContains(t, err, "nodeAddress")
	})

	t.Run("ipv6 brackets", func(t *testing.T) {
		t.Parallel()

		ln, err := tls.Listen("tcp", "[::1]:0", serverTLSConfig(ca, ca.issue(t, "server", true), tls.NoClientCert))
		if err != nil {
			t.Skip("ipv6 loopback not available")
		}

		t.Cleanup(func() { _ = ln.Close() })

		resCh, connCh := serveHandshake(ln)

		cfg := &BlockSyncerConfig{
			NodeAddress: "[::1]:" + portOf(t, ln),
			TLS:         clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
		}

		conn, err := DialNode(cfg, 2*time.Second)
		require.NoError(t, err)

		defer conn.Close()

		res := <-resCh
		require.NoError(t, res.err)
		require.Equal(t, "", res.state.ServerName) // IP literals are never sent as SNI
		require.True(t, conn.(*tls.Conn).ConnectionState().HandshakeComplete)

		(<-connCh).Close()
	})
}

func TestTLSConfig_ClientConfig_Errors(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	other := newTestCA(t)
	dir := t.TempDir()
	client := ca.issue(t, "client", false)
	otherClient := other.issue(t, "client2", false)

	good := clientTLS(t, dir, ca, client)
	noPEM := writeFile(t, dir, "garbage.pem", []byte("not a certificate"))
	mismatchKey := writeFile(t, dir, "other.key", otherClient.keyPEM)
	missing := filepath.Join(dir, "missing.key")

	tests := []struct {
		name   string
		cfg    TLSConfig
		server string
		want   string
	}{
		{"empty caFile", TLSConfig{CertFile: good.CertFile, KeyFile: good.KeyFile}, "a", "tls caFile"},
		{"empty certFile", TLSConfig{CAFile: good.CAFile, KeyFile: good.KeyFile}, "a", "tls certFile"},
		{"empty keyFile", TLSConfig{CAFile: good.CAFile, CertFile: good.CertFile}, "a", "tls keyFile"},
		{"ca without pem", TLSConfig{CAFile: noPEM, CertFile: good.CertFile, KeyFile: good.KeyFile}, "a", "tls caFile"},
		{"missing ca", TLSConfig{CAFile: missing, CertFile: good.CertFile, KeyFile: good.KeyFile}, "a", "tls caFile"},
		{"missing key", TLSConfig{CAFile: good.CAFile, CertFile: good.CertFile, KeyFile: missing}, "a", "tls keyFile"},
		{"mismatched pair", TLSConfig{CAFile: good.CAFile, CertFile: good.CertFile, KeyFile: mismatchKey}, "a", "tls certFile"},
		{"no server name", TLSConfig{CAFile: good.CAFile, CertFile: good.CertFile, KeyFile: good.KeyFile}, "", "serverName"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := tc.cfg.ClientConfig(tc.server)
			require.Error(t, err)
			require.Nil(t, cfg)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestTLSConfig_ClientConfig_Settings(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	cfg := clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false))

	res, err := cfg.ClientConfig("fallback.test")
	require.NoError(t, err)
	require.Equal(t, uint16(tls.VersionTLS13), res.MinVersion)
	require.False(t, res.InsecureSkipVerify)
	require.Nil(t, res.VerifyPeerCertificate)
	require.Equal(t, "fallback.test", res.ServerName)
	require.Len(t, res.Certificates, 1)
	require.NotNil(t, res.RootCAs)

	subjects := res.RootCAs.Subjects() //nolint:staticcheck
	require.Len(t, subjects, 1)
	require.Equal(t, ca.cert.RawSubject, subjects[0])

	cfg.ServerName = "explicit.test"

	res, err = cfg.ClientConfig("fallback.test")
	require.NoError(t, err)
	require.Equal(t, "explicit.test", res.ServerName)
}

func newTLSSyncer(cfg *BlockSyncerConfig) *BlockSyncerImpl {
	return NewBlockSyncer(cfg, indexer.NewBlockSyncerHandlerMock(existingPointSlot, existingPointHashStr),
		hclog.NewNullLogger())
}

func TestSyncer_TLSConnectionClosedOnSetupError(t *testing.T) {
	t.Parallel()

	ca := newTestCA(t)
	ln := listenTLS(t, serverTLSConfig(ca, ca.issue(t, "server", true), tls.RequireAndVerifyClientCert))
	readCh := make(chan error, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			readCh <- err

			return
		}

		defer conn.Close()

		buf := make([]byte, 1024)

		for {
			if _, err := conn.Read(buf); err != nil {
				readCh <- err

				return
			}
		}
	}()

	syncer := newTLSSyncer(&BlockSyncerConfig{
		NetworkMagic: 0, // makes NewConnection fail before the muxer exists
		NodeAddress:  "127.0.0.1:" + portOf(t, ln),
		TLS:          clientTLS(t, t.TempDir(), ca, ca.issue(t, "client", false)),
	})
	syncer.config.TLS.ServerName = testServerName

	defer syncer.Close()

	require.Error(t, syncer.syncExecute())

	select {
	case err := <-readCh:
		require.ErrorIs(t, err, io.EOF)
	case <-time.After(2 * time.Second):
		t.Fatal("server side connection was not closed")
	}
}

func TestSyncer_TLSClientCertRejected(t *testing.T) {
	t.Parallel()

	ca, otherCA := newTestCA(t), newTestCA(t)
	ln := listenTLS(t, serverTLSConfig(ca, ca.issue(t, "server", true), tls.RequireAndVerifyClientCert))
	resCh, connCh := serveHandshake(ln)

	syncer := newTLSSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  "127.0.0.1:" + portOf(t, ln),
		// CA is trusted, but the client certificate is signed by another CA
		TLS: clientTLS(t, t.TempDir(), ca, otherCA.issue(t, "intruder", false)),
	})

	defer syncer.Close()

	errCh := make(chan error, 1)

	go func() { errCh <- syncer.syncExecute() }()

	select {
	case err := <-errCh:
		require.Error(t, err)
	case <-time.After(6 * time.Second):
		t.Fatal("syncExecute hangs")
	}

	require.Error(t, (<-resCh).err)
	require.Nil(t, syncer.connection)

	// server-side conn is closed: a read must fail
	srvConn := <-connCh
	_ = srvConn.SetReadDeadline(time.Now().UTC().Add(time.Second))
	_, err := srvConn.Read(make([]byte, 1))
	require.Error(t, err)
	require.NoError(t, srvConn.Close())
}

func TestBlockSyncerConfig_JSONWithoutTLS(t *testing.T) {
	t.Parallel()

	const in = `{"networkMagic":1,"nodeAddress":"localhost:3001","restartOnError":true,` +
		`"restartDelay":1000,"syncStartTries":3,"keepAlive":true}`

	var cfg BlockSyncerConfig

	require.NoError(t, json.Unmarshal([]byte(in), &cfg))
	require.Nil(t, cfg.TLS)

	out, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.JSONEq(t, in, string(out))
	require.NotContains(t, string(out), "tls")
}
