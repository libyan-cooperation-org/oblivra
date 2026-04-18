package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// FIDO2Credential represents a stored FIDO2/WebAuthn credential
type FIDO2Credential struct {
	ID              string    `json:"id"`
	CredentialID    []byte    `json:"credential_id"`
	PublicKey       []byte    `json:"public_key"`
	SignCount       uint32    `json:"sign_count"`
	DeviceName      string    `json:"device_name"`
	CreatedAt       string    `json:"created_at"`
	LastUsedAt      string    `json:"last_used_at"`
	AttestationType string    `json:"attestation_type"`
}

// FIDO2Challenge represents an authentication challenge
type FIDO2Challenge struct {
	ID        string    `json:"id"`
	Challenge []byte    `json:"challenge"`
	RPID      string    `json:"rp_id"`
	UserID    []byte    `json:"user_id"`
	CreatedAt string    `json:"created_at"`
	ExpiresAt string    `json:"expires_at"`
}

// FIDO2Manager handles WebAuthn/FIDO2 operations
type FIDO2Manager struct {
	mu          sync.RWMutex
	credentials map[string]*FIDO2Credential
	challenges  map[string]*FIDO2Challenge
	rpID        string
	rpName      string
}

func NewFIDO2Manager() *FIDO2Manager {
	return &FIDO2Manager{
		credentials: make(map[string]*FIDO2Credential),
		challenges:  make(map[string]*FIDO2Challenge),
		rpID:        "oblivrashell",
		rpName:      "OblivraShell",
	}
}

// BeginRegistration starts the FIDO2 registration ceremony
func (m *FIDO2Manager) BeginRegistration(userID string, userName string) (*FIDO2Challenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("generate challenge: %w", err)
	}

	uid := sha256.Sum256([]byte(userID))

	c := &FIDO2Challenge{
		ID:        base64.URLEncoding.EncodeToString(challenge[:8]),
		Challenge: challenge,
		RPID:      m.rpID,
		UserID:    uid[:],
		CreatedAt: time.Now().Format(time.RFC3339),
		ExpiresAt: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	}

	m.challenges[c.ID] = c
	return c, nil
}

// CompleteRegistration finishes the FIDO2 registration
func (m *FIDO2Manager) CompleteRegistration(
	challengeID string,
	credentialID []byte,
	publicKey []byte,
	deviceName string,
	attestationType string,
) (*FIDO2Credential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	challenge, ok := m.challenges[challengeID]
	if !ok {
		return nil, fmt.Errorf("challenge not found")
	}

	if time.Now().After(parseTime(challenge.ExpiresAt)) {
		delete(m.challenges, challengeID)
		return nil, fmt.Errorf("challenge expired")
	}

	delete(m.challenges, challengeID)

	cred := &FIDO2Credential{
		ID:              base64.URLEncoding.EncodeToString(credentialID[:8]),
		CredentialID:    credentialID,
		PublicKey:       publicKey,
		SignCount:       0,
		DeviceName:      deviceName,
		CreatedAt:       time.Now().Format(time.RFC3339),
		LastUsedAt:      time.Now().Format(time.RFC3339),
		AttestationType: attestationType,
	}

	m.credentials[cred.ID] = cred
	return cred, nil
}

// BeginAuthentication starts the FIDO2 authentication ceremony
func (m *FIDO2Manager) BeginAuthentication() (*FIDO2Challenge, []FIDO2Credential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.credentials) == 0 {
		return nil, nil, fmt.Errorf("no FIDO2 credentials registered")
	}

	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, nil, fmt.Errorf("generate challenge: %w", err)
	}

	c := &FIDO2Challenge{
		ID:        base64.URLEncoding.EncodeToString(challenge[:8]),
		Challenge: challenge,
		RPID:      m.rpID,
		CreatedAt: time.Now().Format(time.RFC3339),
		ExpiresAt: time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	}

	m.challenges[c.ID] = c

	// Return allowed credentials
	creds := make([]FIDO2Credential, 0, len(m.credentials))
	for _, cred := range m.credentials {
		creds = append(creds, *cred)
	}

	return c, creds, nil
}

// CompleteAuthentication verifies the FIDO2 assertion
func (m *FIDO2Manager) CompleteAuthentication(
	challengeID string,
	credentialID []byte,
	signature []byte,
	authenticatorData []byte,
	clientDataJSON []byte,
	newSignCount uint32,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	challenge, ok := m.challenges[challengeID]
	if !ok {
		return fmt.Errorf("challenge not found")
	}

	if time.Now().After(parseTime(challenge.ExpiresAt)) {
		delete(m.challenges, challengeID)
		return fmt.Errorf("challenge expired")
	}

	delete(m.challenges, challengeID)

	// Find matching credential
	var cred *FIDO2Credential
	credIDStr := base64.URLEncoding.EncodeToString(credentialID[:8])
	for _, c := range m.credentials {
		if c.ID == credIDStr {
			cred = c
			break
		}
	}

	if cred == nil {
		return fmt.Errorf("credential not found")
	}

	// Verify signature (simplified — production needs full WebAuthn verification)
	clientDataHash := sha256.Sum256(clientDataJSON)
	verificationData := append(authenticatorData, clientDataHash[:]...)

	pubKey, err := parsePublicKey(cred.PublicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	hash := sha256.Sum256(verificationData)
	if !ecdsa.VerifyASN1(pubKey, hash[:], signature) {
		return fmt.Errorf("signature verification failed")
	}

	// Check sign count (replay protection)
	if newSignCount > 0 && newSignCount <= cred.SignCount {
		return fmt.Errorf("possible credential cloning detected (sign count mismatch)")
	}

	// Update credential
	cred.SignCount = newSignCount
	cred.LastUsedAt = time.Now().Format(time.RFC3339)

	return nil
}

// ListCredentials returns all registered credentials
func (m *FIDO2Manager) ListCredentials() []FIDO2Credential {
	m.mu.RLock()
	defer m.mu.RUnlock()

	creds := make([]FIDO2Credential, 0, len(m.credentials))
	for _, c := range m.credentials {
		creds = append(creds, *c)
	}
	return creds
}

// RemoveCredential removes a FIDO2 credential
func (m *FIDO2Manager) RemoveCredential(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.credentials[id]; !ok {
		return fmt.Errorf("credential %s not found", id)
	}

	delete(m.credentials, id)
	return nil
}

// HasCredentials returns true if any FIDO2 credentials are registered
func (m *FIDO2Manager) HasCredentials() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.credentials) > 0
}

func parsePublicKey(data []byte) (*ecdsa.PublicKey, error) {
	// Parse COSE public key format
	// In production, use a proper CBOR/COSE parser
	x, y := elliptic.Unmarshal(elliptic.P256(), data)
	if x == nil {
		return nil, fmt.Errorf("invalid public key")
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}, nil
}

