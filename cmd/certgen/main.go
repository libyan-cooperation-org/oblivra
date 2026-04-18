package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kingknull/oblivrashell/internal/security"
)

func main() {
	home, _ := os.UserHomeDir()
	appDir := filepath.Join(home, ".oblivrashell")
	os.MkdirAll(appDir, 0700)

	// 1. Generate Root CA if missing
	caCertPath := filepath.Join(appDir, "ca.pem")
	caKeyPath := filepath.Join(appDir, "ca.key")

	var caPair *security.CertificatePair
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		fmt.Println("Generating Root CA...")
		caPair, err = security.GenerateRootCA()
		if err != nil {
			log.Fatal(err)
		}
		security.SavePair(caPair, caCertPath, caKeyPath)
	} else {
		fmt.Println("Existing Root CA found.")
		certPEM, _ := os.ReadFile(caCertPath)
		keyPEM, _ := os.ReadFile(caKeyPath)
		caPair = &security.CertificatePair{CertPEM: certPEM, KeyPEM: keyPEM}
	}

	// 2. Generate TLS Cert for API
	fmt.Println("Generating API TLS Certificate...")
	tlsPair, err := security.GenerateCert(caPair, "127.0.0.1", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
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

	fmt.Println("Done! Certificates generated in", appDir)
}
