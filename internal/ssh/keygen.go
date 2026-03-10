package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// GenerateED25519Keypair creates a new ED25519 private/public keypair in PEM format
func GenerateED25519Keypair() ([]byte, []byte, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	// Encode private key to PEM
	privBytes, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private key: %w", err)
	}
	privBlock := pem.EncodeToMemory(privBytes)

	// Encode public key to authorized_keys format
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, nil, fmt.Errorf("new public key: %w", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)

	return privBlock, pubBytes, nil
}
