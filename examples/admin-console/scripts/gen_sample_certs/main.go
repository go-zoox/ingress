// Generate sample TLS certs for examples/admin-console.
//
//	go run ./examples/admin-console/scripts/gen_sample_certs
package main

import (
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

type spec struct {
	stem string
	cn   string
	days int
}

func main() {
	root := filepath.Join("examples", "admin-console", "certs")
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	specs := []spec{
		{"api.example.com", "api.example.com", 120},
		{"cdn.example.com", "cdn.example.com", 90},
		{"assets.cdn.example.com", "assets.cdn.example.com", 12},
		{"admin.internal", "admin.internal", 60},
		{"legacy.example.com", "legacy.example.com", -14},
		{"tunnel-a.inlets.example.com", "tunnel-a.inlets.example.com", 200},
		{"waf-demo.example.com", "waf-demo.example.com", 8},
		{"portal.example.com", "portal.example.com", 45},
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		panic(err)
	}
	for _, s := range specs {
		if err := writeCert(root, s); err != nil {
			panic(err)
		}
		fmt.Printf("  %s (%dd)\n", s.stem, s.days)
	}
	fmt.Println("done:", root)
}

func writeCert(root string, s spec) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	now := time.Now()
	notBefore := now.Add(-24 * time.Hour)
	notAfter := now.Add(time.Duration(s.days) * 24 * time.Hour)
	if s.days < 0 {
		notAfter = now.Add(time.Duration(s.days) * 24 * time.Hour)
		notBefore = notAfter.Add(-365 * 24 * time.Hour)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   s.cn,
			Organization: []string{"Ingress Sample CA"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:   []string{s.cn},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	certPath := filepath.Join(root, s.stem+".pem")
	keyPath := filepath.Join(root, s.stem+".key.pem")
	if err := writePEM(certPath, "CERTIFICATE", der); err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	return writePEM(keyPath, "PRIVATE KEY", keyDER)
}

func writePEM(path, typ string, der []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: typ, Bytes: der})
}
