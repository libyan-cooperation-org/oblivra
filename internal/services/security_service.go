package services

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/ssh"
)

// SecurityService exposes security features to the frontend
type SecurityService struct {
	BaseService
	ctx          context.Context
	fido2Manager FIDO2Provider
	ykManager    YubiKeyProvider
	certManager  CertificateProvider
	bus          *eventbus.Bus
	log          *logger.Logger
}

func (s *SecurityService) Name() string { return "security-service" }

// Dependencies returns service dependencies.
func (s *SecurityService) Dependencies() []string {
	return []string{"vault"}
}

// NewSecurityService creates the Wails binding for FIDO2 and YubiKey functionality
func NewSecurityService(fido2 FIDO2Provider, yk YubiKeyProvider, certMgr CertificateProvider, bus *eventbus.Bus, log *logger.Logger) *SecurityService {
	return &SecurityService{
		fido2Manager: fido2,
		ykManager:    yk,
		certManager:  certMgr,
		bus:          bus,
		log:          log.WithPrefix("security_service"),
	}
}

func (s *SecurityService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *SecurityService) Stop(ctx context.Context) error {
	return nil
}

// FIDO2 methods

// FIDO2BeginRegistration starts the process
func (s *SecurityService) FIDO2BeginRegistration(userID, userName string) (*security.FIDO2Challenge, error) {
	return s.fido2Manager.BeginRegistration(userID, userName)
}

// FIDO2ListCredentials returns enrolled keys
func (s *SecurityService) FIDO2ListCredentials() []security.FIDO2Credential {
	return s.fido2Manager.ListCredentials()
}

// FIDO2RemoveCredential removes an enrolled key
func (s *SecurityService) FIDO2RemoveCredential(id string) error {
	return s.fido2Manager.RemoveCredential(id)
}

// FIDO2HasCredentials checks if any keys are enrolled
func (s *SecurityService) FIDO2HasCredentials() bool {
	return s.fido2Manager.HasCredentials()
}

// YubiKey methods

// YubiKeyDetect lists connected YubiKeys
func (s *SecurityService) YubiKeyDetect() ([]security.YubiKeyInfo, error) {
	return s.ykManager.Detect()
}

// YubiKeyGenerateSSHKey provisions a new SSH key on the YubiKey
func (s *SecurityService) YubiKeyGenerateSSHKey(serial, slot, pin string) (string, error) {
	return s.ykManager.GenerateSSHKey(serial, slot, pin)
}

// YubiKeyGetSSHPublicKey gets the public key material
func (s *SecurityService) YubiKeyGetSSHPublicKey(serial, slot string) (string, error) {
	return s.ykManager.GetSSHPublicKey(serial, slot)
}

// YubiKeyDeriveVaultKey performs a challenge-response with the YubiKey to get the hardware key segment
func (s *SecurityService) YubiKeyDeriveVaultKey(serial string, password string) ([]byte, error) {
	return s.ykManager.DeriveVaultKey(serial, password)
}

// SSH Certificate methods

func (s *SecurityService) SSHListCertificates() ([]ssh.CertificateInfo, error) {
	return s.certManager.ListCertificates()
}

// CheckSSHCertExpiry returns certs expiring in a certain number of hours
func (s *SecurityService) CheckSSHCertExpiry(hours int) ([]ssh.CertificateInfo, error) {
	importTime := time.Duration(hours) * time.Hour
	return s.certManager.CheckExpiry(importTime)
}
