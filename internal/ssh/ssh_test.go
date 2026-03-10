package ssh

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 22 {
		t.Errorf("expected port 22, got %d", cfg.Port)
	}
	if cfg.TermType != "xterm-256color" {
		t.Errorf("expected xterm-256color, got %s", cfg.TermType)
	}
	if cfg.ConnectTimeout.Seconds() != 10 {
		t.Errorf("expected 10s timeout, got %v", cfg.ConnectTimeout)
	}
	if cfg.AuthMethod != AuthPublicKey {
		t.Errorf("expected public key auth, got %s", cfg.AuthMethod)
	}
	if !cfg.Compression {
		t.Error("expected compression enabled by default")
	}
}

func TestConnectionConfigAddress(t *testing.T) {
	cfg := ConnectionConfig{Host: "example.com", Port: 2222}
	expected := "example.com:2222"
	if cfg.Address() != expected {
		t.Errorf("expected %s, got %s", expected, cfg.Address())
	}
}

func TestNewSessionManager(t *testing.T) {
	mgr := NewSessionManager(10)

	if mgr.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions, got %d", mgr.ActiveCount())
	}

	cfg := DefaultConfig()
	cfg.Host = "test.example.com"
	cfg.Username = "testuser"

	session := NewSession("host-1", "Test Host", cfg)
	if err := mgr.Add(session); err != nil {
		t.Fatalf("failed to add session: %v", err)
	}

	retrieved, ok := mgr.Get(session.ID)
	if !ok {
		t.Fatal("session not found")
	}
	if retrieved.ID != session.ID {
		t.Errorf("session ID mismatch: got %s, want %s", retrieved.ID, session.ID)
	}

	mgr.Remove(session.ID)
	_, ok = mgr.Get(session.ID)
	if ok {
		t.Error("session should have been removed")
	}
}

func TestSessionManagerMaxSessions(t *testing.T) {
	mgr := NewSessionManager(2)

	cfg := DefaultConfig()
	cfg.Host = "test.example.com"
	cfg.Username = "testuser"

	s1 := NewSession("h1", "Host 1", cfg)
	s2 := NewSession("h2", "Host 2", cfg)
	s3 := NewSession("h3", "Host 3", cfg)

	mgr.Add(s1)
	mgr.Add(s2)

	if err := mgr.Add(s3); err == nil {
		t.Error("expected max sessions error, got nil")
	}
}

func TestDefaultSessionManagerLimit(t *testing.T) {
	mgr := NewSessionManager(0) // should default to 50
	if mgr.maxSessions != 50 {
		t.Errorf("expected default max 50, got %d", mgr.maxSessions)
	}
}

func TestParseSSHConfig(t *testing.T) {
	entries, err := ParseSSHConfig()
	if err != nil {
		t.Skipf("SSH config not found: %v", err)
	}
	t.Logf("Found %d SSH config entries", len(entries))
	for _, e := range entries {
		t.Logf("  %s -> %s:%d (user: %s)", e.Alias, e.Hostname, e.Port, e.User)
	}
}

func TestTunnelConfig(t *testing.T) {
	cfg := TunnelConfig{
		Type:       TunnelLocal,
		LocalHost:  "127.0.0.1",
		LocalPort:  8080,
		RemoteHost: "localhost",
		RemotePort: 80,
	}
	if cfg.Type != TunnelLocal {
		t.Errorf("expected local tunnel type, got %s", cfg.Type)
	}
	if cfg.LocalHost != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", cfg.LocalHost)
	}
	if cfg.LocalPort != 8080 {
		t.Errorf("expected 8080, got %d", cfg.LocalPort)
	}
	if cfg.RemoteHost != "localhost" {
		t.Errorf("expected localhost, got %s", cfg.RemoteHost)
	}
	if cfg.RemotePort != 80 {
		t.Errorf("expected 80, got %d", cfg.RemotePort)
	}
}

func TestNewSession(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Host = "host.example.com"
	cfg.Username = "user"

	s := NewSession("host-id-1", "My Server", cfg)
	if s.ID == "" {
		t.Error("session ID should not be empty")
	}
	if s.HostID != "host-id-1" {
		t.Errorf("unexpected host ID: %s", s.HostID)
	}
	if s.Status != SessionActive {
		t.Errorf("expected active status, got %s", s.Status)
	}
}
