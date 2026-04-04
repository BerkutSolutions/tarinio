package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// DevelopmentLetsEncryptClient is a deterministic local ACME stand-in for Stage 1 wiring.
type DevelopmentLetsEncryptClient struct{}

func NewDevelopmentLetsEncryptClient() *DevelopmentLetsEncryptClient {
	return &DevelopmentLetsEncryptClient{}
}

func (c *DevelopmentLetsEncryptClient) Issue(commonName string, sanList []string, _ *ACMEIssueOptions) (IssuedMaterial, error) {
	return c.issueMaterial(commonName, sanList, 90*24*time.Hour)
}

func (c *DevelopmentLetsEncryptClient) Renew(commonName string, sanList []string, _ *ACMEIssueOptions) (IssuedMaterial, error) {
	return c.issueMaterial(commonName, sanList, 90*24*time.Hour)
}

func (c *DevelopmentLetsEncryptClient) issueMaterial(commonName string, sanList []string, validity time.Duration) (IssuedMaterial, error) {
	commonName = strings.TrimSpace(commonName)
	if commonName == "" {
		return IssuedMaterial{}, fmt.Errorf("common name is required")
	}

	now := time.Now().UTC()
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return IssuedMaterial{}, err
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return IssuedMaterial{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	addSAN := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if ip := net.ParseIP(value); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			return
		}
		template.DNSNames = append(template.DNSNames, value)
	}
	addSAN(commonName)
	for _, item := range sanList {
		addSAN(item)
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return IssuedMaterial{}, err
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	privateKeyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return IssuedMaterial{}, err
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyDER})

	return IssuedMaterial{
		CertificatePEM: certificatePEM,
		PrivateKeyPEM:  privateKeyPEM,
		NotBefore:      template.NotBefore.Format(time.RFC3339),
		NotAfter:       template.NotAfter.Format(time.RFC3339),
	}, nil
}
