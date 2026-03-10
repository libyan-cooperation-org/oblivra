package vault

// Provider defines the interface for secure data storage and key management.
type Provider interface {
	IsSetup() bool
	Setup(password string, yubiKeySerial string) error
	SetupWithTPM(password string, yubiKeySerial string, pcr int) error
	Unlock(password string, hardwareKey []byte, rememberMe bool) error
	UnlockWithKeychain() error
	IsUnlocked() bool
	Lock()
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	AccessMasterKey(fn func(key []byte) error) error
	GetYubiKeySerial() string
	IsTPMBound() bool
	GetPassword(id string) ([]byte, error)
	GetPrivateKey(id string) ([]byte, string, error)
}
