package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Signer holds the agent's ed25519 keypair and signs every outbound event
// before forwarding. This puts cryptographic integrity *at the edge* — an
// MITM that decrypts the TLS channel still can't alter individual events
// without invalidating the signature, because the agent's signing key
// never leaves the host.
//
// The keypair is generated on first run, stored at <stateDir>/agent.ed25519
// (private; mode 0600) and <stateDir>/agent.ed25519.pub (operator copies
// this to the platform's `OBLIVRA_AGENT_PUBKEYS` allow-list).
//
// Each signed event grows by ~110 bytes (signature base64 + key fingerprint).
// At sustained 5k EPS that's ~550KB/s extra — acceptable on modern wire.
type Signer struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	id   string // base64 of pubkey[:8] — short fingerprint for human use
}

// LoadOrCreateSigner reads an existing keypair or generates one.
func LoadOrCreateSigner(stateDir string) (*Signer, error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	privPath := filepath.Join(stateDir, "agent.ed25519")
	pubPath := privPath + ".pub"

	if body, err := os.ReadFile(privPath); err == nil && len(body) == ed25519.PrivateKeySize {
		priv := ed25519.PrivateKey(body)
		pub := priv.Public().(ed25519.PublicKey)
		return &Signer{priv: priv, pub: pub, id: shortID(pub)}, nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return nil, fmt.Errorf("write priv key: %w", err)
	}
	if err := os.WriteFile(pubPath,
		[]byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write pub key: %w", err)
	}
	return &Signer{priv: priv, pub: pub, id: shortID(pub)}, nil
}

// PublicKeyB64 returns the base64-encoded public key (so the operator can
// drop it into the platform's allow-list).
func (s *Signer) PublicKeyB64() string {
	return base64.StdEncoding.EncodeToString(s.pub)
}

// FingerprintShort is what we stamp into the event provenance.
func (s *Signer) FingerprintShort() string { return s.id }

// SignEvent takes a marshalled event JSON, computes a deterministic hash
// over its canonicalised view, signs it, and returns the augmented JSON
// with `agentSig` + `agentKeyId` added. The augmented event still parses
// as a normal event on the server side — extra fields are stripped or
// preserved depending on receiver config.
func (s *Signer) SignEvent(rawJSON string) (string, error) {
	if s == nil || s.priv == nil {
		return rawJSON, nil
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &doc); err != nil {
		return "", err
	}
	// Build a canonicalised payload (sorted keys, signature fields removed)
	// so verification is reproducible.
	delete(doc, "agentSig")
	delete(doc, "agentKeyId")
	canon, err := canonJSON(doc)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(s.priv, canon)
	doc["agentSig"] = base64.StdEncoding.EncodeToString(sig)
	doc["agentKeyId"] = s.id
	out, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Verify is the server-side counterpart, exposed here because some
// integrations (test harness, dry-run) want to verify locally too.
func Verify(pubB64 string, signedJSON []byte) error {
	pubBytes, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		return err
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return errors.New("bad pubkey size")
	}
	var doc map[string]any
	if err := json.Unmarshal(signedJSON, &doc); err != nil {
		return err
	}
	sigB64, _ := doc["agentSig"].(string)
	if sigB64 == "" {
		return errors.New("no signature on event")
	}
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return err
	}
	delete(doc, "agentSig")
	delete(doc, "agentKeyId")
	canon, err := canonJSON(doc)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), canon, sig) {
		return errors.New("signature does not verify")
	}
	return nil
}

// canonJSON is a deterministic JSON encoder — keys at every level sorted
// alphabetically. This is what makes Sign + Verify stable across
// json package implementations and Go versions.
func canonJSON(v any) ([]byte, error) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf := []byte{'{'}
		for i, k := range keys {
			if i > 0 {
				buf = append(buf, ',')
			}
			kb, _ := json.Marshal(k)
			buf = append(buf, kb...)
			buf = append(buf, ':')
			vb, err := canonJSON(t[k])
			if err != nil {
				return nil, err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, '}')
		return buf, nil
	case []any:
		buf := []byte{'['}
		for i, item := range t {
			if i > 0 {
				buf = append(buf, ',')
			}
			ib, err := canonJSON(item)
			if err != nil {
				return nil, err
			}
			buf = append(buf, ib...)
		}
		buf = append(buf, ']')
		return buf, nil
	default:
		return json.Marshal(t)
	}
}

func shortID(pub ed25519.PublicKey) string {
	if len(pub) < 8 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(pub[:8])
}
