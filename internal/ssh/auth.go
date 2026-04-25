package ssh

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// InsecureHostKeyWarning is emitted to the audit log whenever a connection is
// established with host key verification disabled (SA-01).
const InsecureHostKeyWarning = "[SECURITY WARNING] SSH host key verification is DISABLED. " +
	"This connection is vulnerable to man-in-the-middle attacks. " +
	"Enable StrictHostKey in host settings for production use."

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
	// SA-06: sshAgentAuth now returns the underlying conn so we can close it after use.
	if cfg.EnableAgent {
		if agentAuth, conn := sshAgentAuth(); agentAuth != nil {
			methods = append(methods, sshAgentAuthWithCleanup(agentAuth, conn))
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

// sshAgentAuth returns an SSH agent auth method and the underlying net.Conn so the
// caller can close it when authentication is complete (SA-06: fix fd leak).
// Returns (nil, nil) when no SSH agent is available.
func sshAgentAuth() (ssh.AuthMethod, net.Conn) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, nil
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), conn
}

// sshAgentAuthWithCleanup wraps the agent auth callback so that the underlying
// Unix socket is closed after the signers are retrieved (SA-06).
func sshAgentAuthWithCleanup(method ssh.AuthMethod, conn net.Conn) ssh.AuthMethod {
	if conn == nil {
		return method
	}
	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		defer conn.Close()
		return agentClient.Signers()
	})
}

// buildHostKeyCallback creates host key verification.
// SA-01: when strict=false, emit a prominent security warning and log each connection.
func buildHostKeyCallback(strict bool) (ssh.HostKeyCallback, error) {
	if !strict {
		log.Println(InsecureHostKeyWarning)
		//nolint:gosec // caller explicitly disabled strict host key checking
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			log.Printf("[SECURITY] InsecureIgnoreHostKey: connected to %s (%s) — no host key verification",
				hostname, remote.String())
			return nil
		}, nil
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

	// SA-03: id_dsa removed — DSA is deprecated (FIPS 186-5 withdrawn) and cryptographically weak.
	keyNames := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
	for _, name := range keyNames {
		keyPath := filepath.Join(home, ".ssh", name)
		data, err := os.ReadFile(keyPath)
		if err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("no default SSH keys found")
}
