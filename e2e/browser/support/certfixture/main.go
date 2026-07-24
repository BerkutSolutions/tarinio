package main

import (
  "archive/zip"
  "crypto/rand"
  "crypto/rsa"
  "crypto/x509"
  "crypto/x509/pkix"
  "encoding/pem"
  "fmt"
  "math/big"
  "os"
  "path/filepath"
  "time"
)

func main() {
  if len(os.Args) != 3 { panic("usage: certfixture OUTPUT_DIR CERTIFICATE_ID") }
  dir, id := os.Args[1], os.Args[2]
  if err := os.MkdirAll(dir, 0o700); err != nil { panic(err) }
  key, err := rsa.GenerateKey(rand.Reader, 2048); if err != nil { panic(err) }
  serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120)); if err != nil { panic(err) }
  now := time.Now().UTC()
  tmpl := x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: id + ".example.test"}, DNSNames: []string{id + ".example.test"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24*time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
  der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key); if err != nil { panic(err) }
  cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
  privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
  certPath, keyPath, zipPath := filepath.Join(dir, "certificate.pem"), filepath.Join(dir, "private.key"), filepath.Join(dir, "materials.zip")
  if err := os.WriteFile(certPath, cert, 0o600); err != nil { panic(err) }
  if err := os.WriteFile(keyPath, privateKey, 0o600); err != nil { panic(err) }
  file, err := os.Create(zipPath); if err != nil { panic(err) }
  zw := zip.NewWriter(file)
  for name, content := range map[string][]byte{id+"/certificate.pem": cert, id+"/private.key": privateKey} { w, err := zw.Create(name); if err != nil { panic(err) }; if _, err = w.Write(content); err != nil { panic(err) } }
  if err := zw.Close(); err != nil { panic(err) }; if err := file.Close(); err != nil { panic(err) }
  fmt.Printf("%s\n%s\n%s\n", certPath, keyPath, zipPath)
}
