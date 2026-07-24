package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"
)

type materials struct{ ca, caKey, clientCert, clientKey []byte }

func main() {
	ca, caKey, serverCert, clientCert, clientKey, err := issueMaterials()
	if err != nil {
		log.Fatal(err)
	}
	m := materials{ca: ca, caKey: caKey, clientCert: clientCert, clientKey: clientKey}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(ca) {
		log.Fatal("parse generated CA")
	}
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /__e2e/ca.pem", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(m.ca) })
		mux.HandleFunc("GET /__e2e/ca.key", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(m.caKey) })
		mux.HandleFunc("GET /__e2e/client.pem", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(m.clientCert) })
		mux.HandleFunc("GET /__e2e/client.key", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(m.clientKey) })
		log.Fatal(http.ListenAndServe(":8080", mux))
	}()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-E2E-mTLS", "verified")
		_, _ = w.Write([]byte("mtls-upstream-ok"))
	})
	srv := &http.Server{Addr: ":8443", Handler: h, TLSConfig: &tls.Config{Certificates: []tls.Certificate{serverCert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: caPool, MinVersion: tls.VersionTLS12}}
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func issueMaterials() ([]byte, []byte, tls.Certificate, []byte, []byte, error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, tls.Certificate{}, nil, nil, err
	}
	caTpl := certTemplate("e2e-mtls-ca", true)
	caTpl.IsCA = true
	caTpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, tls.Certificate{}, nil, nil, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})
	issue := func(name string, server bool) ([]byte, []byte, error) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, err
		}
		tpl := certTemplate(name, false)
		tpl.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
		tpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		if server {
			tpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
			tpl.DNSNames = []string{"mtls-upstream"}
			tpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		}
		der, err := x509.CreateCertificate(rand.Reader, tpl, caTpl, &key.PublicKey, caKey)
		if err != nil {
			return nil, nil, err
		}
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), nil
	}
	serverPEM, serverKey, err := issue("mtls-upstream", true)
	if err != nil {
		return nil, nil, tls.Certificate{}, nil, nil, err
	}
	clientPEM, clientKey, err := issue("waf-e2e-client", false)
	if err != nil {
		return nil, nil, tls.Certificate{}, nil, nil, err
	}
	serverCert, err := tls.X509KeyPair(serverPEM, serverKey)
	if err != nil {
		return nil, nil, tls.Certificate{}, nil, nil, err
	}
	return caPEM, caKeyPEM, serverCert, clientPEM, clientKey, nil
}

func certTemplate(cn string, ca bool) *x509.Certificate {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	return &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: cn}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(24 * time.Hour), BasicConstraintsValid: true, IsCA: ca}
}

var _ sync.Once
