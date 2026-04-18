package security

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// CertificateManager handles dynamic loading and rotation of TLS certificates.
// It allows the REST server to pick up new certificates without restarting.
type CertificateManager struct {
	certPath string
	keyPath  string
	log      *logger.Logger
	mu       sync.RWMutex
	cert     *tls.Certificate
}

// NewCertificateManager creates a new certificate manager.
func NewCertificateManager(certPath, keyPath string, log *logger.Logger) *CertificateManager {
	return &CertificateManager{
		certPath: certPath,
		keyPath:  keyPath,
		log:      log.WithPrefix("cert_manager"),
	}
}

// Load attempts to load the certificate from disk.
func (m *CertificateManager) Load() error {
	cert, err := tls.LoadX509KeyPair(m.certPath, m.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load key pair: %w", err)
	}

	m.mu.Lock()
	m.cert = &cert
	m.mu.Unlock()

	m.log.Info("Successfully loaded TLS certificate from %s", m.certPath)
	return nil
}

// GetCertificate returns the current certificate for tls.Config.GetCertificate.
func (m *CertificateManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.cert == nil {
		return nil, fmt.Errorf("no certificate loaded")
	}

	return m.cert, nil
}

// Reload forces a reload of the certificate from disk.
func (m *CertificateManager) Reload() error {
	m.log.Info("Reloading TLS certificate...")
	return m.Load()
}
