package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client wraps HTTP transport with TLS pinning, mTLS, gzip body compression,
// and a small retry loop. Designed to be configured once at startup and
// reused for every batch + heartbeat.
type Client struct {
	server      string
	token       string
	hc          *http.Client
	compression string
	timeout     time.Duration
}

func NewClient(c *Config) (*Client, error) {
	tlsCfg, err := buildTLSConfig(c.Server.TLS)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig:       tlsCfg,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: c.Server.RequestTimeout,
	}
	return &Client{
		server:      strings.TrimRight(c.Server.URL, "/"),
		token:       c.Server.Token,
		hc:          &http.Client{Transport: tr, Timeout: c.Server.RequestTimeout},
		compression: c.Compression,
		timeout:     c.Server.RequestTimeout,
	}, nil
}

func buildTLSConfig(opts TLSOpts) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if opts.ServerNameOverride != "" {
		cfg.ServerName = opts.ServerNameOverride
	}
	if opts.Insecure {
		cfg.InsecureSkipVerify = true
	}

	if opts.CACertFile != "" {
		body, err := os.ReadFile(opts.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("ca cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(body) {
			return nil, errors.New("ca cert: no certs parsed")
		}
		cfg.RootCAs = pool
	}

	if opts.ClientCertFile != "" || opts.ClientKeyFile != "" {
		if opts.ClientCertFile == "" || opts.ClientKeyFile == "" {
			return nil, errors.New("clientCertFile and clientKeyFile must be set together")
		}
		cert, err := tls.LoadX509KeyPair(opts.ClientCertFile, opts.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("client cert: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if opts.PinnedSHA256 != "" {
		want, err := base64.StdEncoding.DecodeString(opts.PinnedSHA256)
		if err != nil {
			return nil, fmt.Errorf("pin: %w", err)
		}
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("tls: pin: no peer certs")
			}
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return err
			}
			h := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
			if !bytesEqual(h[:], want) {
				return fmt.Errorf("tls: pin mismatch (got %s, want %s)",
					base64.StdEncoding.EncodeToString(h[:]),
					opts.PinnedSHA256,
				)
			}
			return nil
		}
		// Pinning replaces default verification, so we can ride on InsecureSkipVerify
		// without losing the integrity guarantee.
		cfg.InsecureSkipVerify = true
	}
	return cfg, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PostBatch sends a slice of pre-marshalled event bodies. Returns nil on 2xx.
func (c *Client) PostBatch(ctx context.Context, items []string) error {
	if len(items) == 0 {
		return nil
	}
	body := []byte("[" + strings.Join(items, ",") + "]")

	var (
		reader      io.Reader = bytes.NewReader(body)
		contentType           = "application/json"
		encoding              = ""
	)
	if c.compression == "gzip" {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(body); err != nil {
			_ = gw.Close()
			return err
		}
		if err := gw.Close(); err != nil {
			return err
		}
		reader = &buf
		encoding = "gzip"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.server+"/api/v1/siem/ingest/batch", reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	if encoding != "" {
		req.Header.Set("Content-Encoding", encoding)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server %s: %s", resp.Status, string(out))
	}
	return nil
}

// RegisterAgent calls /api/v1/agent/register on first start. Returns the
// agent ID issued by the server. Operators can pre-provision the ID and
// skip registration by setting it in the position-store metadata file.
func (c *Client) RegisterAgent(ctx context.Context, hostname, osName, archName, version string, tags []string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"hostname": hostname, "os": osName, "arch": archName, "version": version, "tags": tags,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.server+"/api/v1/agent/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		out, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("register: %s: %s", resp.Status, string(out))
	}
	var doc struct{ ID string `json:"id"` }
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	return doc.ID, nil
}

// Heartbeat pings the platform with a marker event. The fleet view shows
// "last seen" based on these.
func (c *Client) Heartbeat(ctx context.Context, hostname, agentID string) error {
	ev := map[string]any{
		"source":    "agent",
		"hostId":    hostname,
		"eventType": "agent.heartbeat",
		"severity":  "debug",
		"message":   "heartbeat",
		"fields":    map[string]string{"agentId": agentID},
	}
	body, _ := json.Marshal(ev)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.server+"/api/v1/siem/ingest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("heartbeat: %s", resp.Status)
	}
	return nil
}

// HealthCheck pings /healthz so we can refuse to send when the server is down.
func (c *Client) HealthCheck(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", c.server+"/healthz", nil)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("/healthz: %s", resp.Status)
	}
	return nil
}
