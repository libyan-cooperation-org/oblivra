package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// buildAuthMethods constructs SSH auth methods from config
func buildAuthMethods(cfg *ConnectionConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	switch cfg.AuthMethod {
	case AuthPassword:
		if len(cfg.Password) == 0 {
			return nil, fmt.Errorf("password required for password auth")
		}
		methods = append(methods, ssh.Password(string(cfg.Password)))

	case AuthPublicKey:
		keyData := cfg.PrivateKey
		if len(keyData) == 0 {
			// Try to load default keys from ~/.ssh
			if data, err := LoadDefaultKeys(); err == nil {
				keyData = data
			}
		}

		signer, err := parsePrivateKey(keyData, cfg.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))

	case AuthKeyboardInteractive:
		methods = append(methods, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = string(cfg.Password)
				}
				return answers, nil
			},
		))

	case AuthCertificate:
		signer, err := parsePrivateKey(cfg.PrivateKey, cfg.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("parse certificate key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))

	default:
		return nil, fmt.Errorf("unsupported auth method: %s", cfg.AuthMethod)
	}

	// Try SSH agent as fallback
	if cfg.EnableAgent {
		if agentAuth := sshAgentAuth(); agentAuth != nil {
			methods = append(methods, agentAuth)
		}
	}

	return methods, nil
}

// parsePrivateKey parses a private key with optional passphrase
func parsePrivateKey(keyData []byte, passphrase []byte) (ssh.Signer, error) {
	if len(keyData) == 0 {
		return nil, fmt.Errorf("empty private key data")
	}

	var signer ssh.Signer
	var err error

	if len(passphrase) > 0 {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, passphrase)
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}

	return signer, nil
}

// sshAgentAuth returns SSH agent auth method if available
func sshAgentAuth() ssh.AuthMethod {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers)
}

// buildHostKeyCallback creates host key verification
func buildHostKeyCallback(strict bool) (ssh.HostKeyCallback, error) {
	if !strict {
		//nolint:gosec // user explicitly disabled strict checking
		return ssh.InsecureIgnoreHostKey(), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			return nil, fmt.Errorf("create .ssh dir: %w", err)
		}
		f, err := os.Create(knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("create known_hosts: %w", err)
		}
		f.Close()
	}

	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("parse known_hosts: %w", err)
	}

	return callback, nil
}

// LoadDefaultKeys attempts to load default SSH keys
func LoadDefaultKeys() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	keyNames := []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"}
	for _, name := range keyNames {
		keyPath := filepath.Join(home, ".ssh", name)
		data, err := os.ReadFile(keyPath)
		if err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("no default SSH keys found")
}
