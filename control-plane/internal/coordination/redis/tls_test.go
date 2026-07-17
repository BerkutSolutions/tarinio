package redis

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewTLSConfigRequiresTrustedCA(t *testing.T) {
	if _, err := newTLSConfig(Config{Addr: "redis:6379", TLS: true}); err == nil {
		t.Fatal("expected TLS without CA to be rejected")
	}
	if _, err := newTLSConfig(Config{Addr: "redis:6379", TLS: true, TLSCAFile: "missing-ca.crt"}); err == nil {
		t.Fatal("expected missing CA to be rejected")
	}
}

func TestClientPingOverTLSWithACLAuthentication(t *testing.T) {
	caPEM, serverCert := testRedisTLSCertificate(t, "redis.test")
	caPath := t.TempDir() + "/ca.crt"
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		t.Fatalf("write CA: %v", err)
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{serverCert}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatalf("listen TLS: %v", err)
	}
	defer listener.Close()
	go serveRedisTLSForTest(t, listener)

	client := NewClient(Config{Addr: listener.Addr().String(), Username: "waf-coordination", Password: "secret", TLS: true, TLSCAFile: caPath, TLSServerName: "redis.test"})
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("TLS Redis ping failed: %v", err)
	}
}

func serveRedisTLSForTest(t *testing.T, listener net.Listener) {
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for _, response := range []string{"+OK\r\n", "+PONG\r\n"} {
		command, err := readRedisTestCommand(reader)
		if err != nil {
			t.Errorf("read redis command: %v", err)
			return
		}
		if len(command) == 0 {
			t.Error("empty redis command")
			return
		}
		if _, err := conn.Write([]byte(response)); err != nil {
			t.Errorf("write redis response: %v", err)
			return
		}
	}
}

func readRedisTestCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "*") {
		return nil, err
	}
	count := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(line), "*%d", &count); err != nil {
		return nil, err
	}
	parts := make([]string, 0, count)
	for range count {
		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		length := 0
		if _, err := fmt.Sscanf(strings.TrimSpace(lengthLine), "$%d", &length); err != nil {
			return nil, err
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		parts = append(parts, string(buf[:length]))
	}
	return parts, nil
}

func testRedisTLSCertificate(t *testing.T, serverName string) ([]byte, tls.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: serverName}, DNSNames: []string{serverName}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return certPEM, cert
}
