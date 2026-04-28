//go:build ignore
// +build ignore

// Scratch script — manual run only:
//   go run scratch/regen_certs.go
// Excluded from `go build ./...` so the package's three main files
// don't fight each other for the symbol `main`.

package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kingknull/oblivrashell/internal/security"
)

func MainRegenCerts() {
	appDir := filepath.Join(os.Getenv("APPDATA"), "sovereign-terminal")
	os.MkdirAll(appDir, 0700)

	fmt.Println("Regenerating Sovereign Certificates in", appDir)

	// 1. Generate Root CA
	caCertPath := filepath.Join(appDir, "ca.pem")
	caKeyPath := filepath.Join(appDir, "ca.key")

	fmt.Println("Generating Root CA...")
	caPair, err := security.GenerateRootCA()
	if err != nil {
		log.Fatal(err)
	}
	security.SavePair(caPair, caCertPath, caKeyPath)

	// 2. Generate TLS Cert for API (covering localhost and machine name)
	fmt.Println("Generating API TLS Certificate...")
	tlsPair, err := security.GenerateCert(caPair, "localhost", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	if err != nil {
		log.Fatal(err)
	}
	security.SavePair(tlsPair, filepath.Join(appDir, "cert.pem"), filepath.Join(appDir, "key.pem"))

	// 3. Generate Code Signing Cert
	fmt.Println("Generating Code Signing Certificate...")
	signPair, err := security.GenerateCert(caPair, "OblivraShell Code Signer", []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning})
	if err != nil {
		log.Fatal(err)
	}
	security.SavePair(signPair, filepath.Join(appDir, "codesign.pem"), filepath.Join(appDir, "codesign.key"))

	fmt.Println("Done! All certificates regenerated.")
}
