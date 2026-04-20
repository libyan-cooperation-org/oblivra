package ssh

import (
	"fmt"
	"time"
)

// AuthMethod defines how to authenticate
type AuthMethod string

const (
	AuthPassword            AuthMethod = "password"
	AuthPublicKey           AuthMethod = "key"
	AuthKeyboardInteractive AuthMethod = "keyboard-interactive"
	AuthCertificate         AuthMethod = "certificate"
)

// ConnectionConfig holds everything needed to establish an SSH connection
type ConnectionConfig struct {
	// Target
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`

	// Authentication
	AuthMethod AuthMethod `json:"auth_method"`
	Password   []byte     `json:"-"` // never serialize
	PrivateKey []byte     `json:"-"` // never serialize
	Passphrase []byte     `json:"-"` // never serialize

	// Jump/Bastion
	JumpHosts []JumpHostConfig `json:"jump_hosts,omitempty"`

	// Timeouts
	ConnectTimeout    time.Duration `json:"connect_timeout"`
	KeepAliveInterval time.Duration `json:"keepalive_interval"`
	KeepAliveMax      int           `json:"keepalive_max"`

	// Terminal
	TermType string `json:"term_type"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`

	// Features
	EnableAgent   bool `json:"enable_agent"`
	StrictHostKey bool `json:"strict_host_key"`
	Compression   bool `json:"compression"`
}

// JumpHostConfig defines a bastion/jump host
type JumpHostConfig struct {
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	Username   string     `json:"username"`
	AuthMethod AuthMethod `json:"auth_method"`
	Password   []byte     `json:"-"`
	PrivateKey []byte     `json:"-"`
	Passphrase []byte     `json:"-"`
}

// Address returns host:port string
func (c *ConnectionConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DefaultConfig returns sensible defaults
func DefaultConfig() ConnectionConfig {
	return ConnectionConfig{
		Port:              22,
		AuthMethod:        AuthPublicKey,
		ConnectTimeout:    10 * time.Second,
		KeepAliveInterval: 30 * time.Second,
		KeepAliveMax:      3,
		TermType:          "xterm-256color",
		Cols:              120,
		Rows:              40,
		EnableAgent:       true, // Enable SSH agent by default
		StrictHostKey:     true,
		Compression:       true,
	}
}
