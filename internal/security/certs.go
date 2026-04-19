package security

// WHY: The security package provides utilities for local certificate management.
// Specifically, it enables bypassing Windows SmartScreen and network alerts by generating
// a local Root CA that the user can trust, allowing for "trusted" local HTTPS and signed binaries.
// Trade-off: While self-signing isn't suitable for global trust, it effectively secures the local
// loopback interface (127.0.0.1) without requiring expensive third-party CA signatures.

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertificatePair holds raw PEM data
type CertificatePair struct {
	CertPEM []byte
	KeyPEM  []byte
}

// GenerateRootCA creates a self-signed Root CA
func GenerateRootCA() (*CertificatePair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"OblivraShell Internal CA"},
			CommonName:   "OblivraShell Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return &CertificatePair{CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

// GenerateCert signs a new certificate using the CA
func GenerateCert(caPair *CertificatePair, commonName string, usages []x509.ExtKeyUsage) (*CertificatePair, error) {
	caCertBlock, _ := pem.Decode(caPair.CertPEM)
	caKeyBlock, _ := pem.Decode(caPair.KeyPEM)

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, err
	}
	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"OblivraShell"},
			CommonName:   commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: usages,
	}

	if commonName == "127.0.0.1" || commonName == "localhost" {
		template.DNSNames = []string{"localhost"}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &priv.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return &CertificatePair{CertPEM: certPEM, KeyPEM: keyPEM}, nil
}

// SavePair writes certs to disk
func SavePair(pair *CertificatePair, certPath, keyPath string) error {
	if err := os.WriteFile(certPath, pair.CertPEM, 0640); err != nil {
		return err
	}
	return os.WriteFile(keyPath, pair.KeyPEM, 0600)
}

// EnsureLocalCerts checks for the existence of certPath and keyPath.
// If missing, it generates a local CA and signs a new certificate for "localhost".
func EnsureLocalCerts(certPath, keyPath string) error {
	caPath := filepath.Join(filepath.Dir(certPath), "root_ca.pem")

	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			if _, err := os.Stat(caPath); err == nil {
				return nil // All exist
			}
		}
	}

	// Missing components, generate new set
	ca, err := GenerateRootCA()
	if err != nil {
		return fmt.Errorf("failed to generate local CA: %w", err)
	}

	// Sign a cert for localhost
	server, err := GenerateCert(ca, "localhost", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth})
	if err != nil {
		return fmt.Errorf("failed to sign local certificate: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		return err
	}

	// Save CA root for clients to trust
	if err := os.WriteFile(caPath, ca.CertPEM, 0640); err != nil {
		return fmt.Errorf("failed to save Root CA: %w", err)
	}

	if err := SavePair(server, certPath, keyPath); err != nil {
		return fmt.Errorf("failed to save local certificates: %w", err)
	}

	return nil
}
