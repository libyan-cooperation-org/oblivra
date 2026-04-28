package tamper

// Heartbeat scheduler — every 30s POSTs an HMAC-signed payload to
// /api/v1/agent/heartbeat carrying:
//
//   uptime_s         — seconds since agent process start
//   log_file_inode   — current inode (Unix) or pseudo-inode (Windows)
//   log_file_size    — current size in bytes
//   last_hash        — head of the local hash chain
//   wall_clock       — agent's UTC clock; server compares against its own
//
// Server-side detection rules fire on:
//   • log_truncated   — log_file_size shrank between heartbeats
//   • log_inode_changed — file rotated (logged but legitimate)
//   • time_skew       — wall_clock differs from server > 60s

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type heartbeatPayload struct {
	AgentID      string `json:"agent_id"`
	UptimeS      int64  `json:"uptime_s"`
	LogFileInode uint64 `json:"log_file_inode"`
	LogFileSize  int64  `json:"log_file_size"`
	LastHash     string `json:"last_hash"`
	WallClock    string `json:"wall_clock"`
}

func (s *Subsystem) runHeartbeat(ctx context.Context) {
	defer s.wg.Done()

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !s.cfg.VerifyTLS},
		},
	}

	// First heartbeat fires immediately so the server's integrity
	// table populates without waiting 30s on a fresh start.
	s.sendHeartbeat(ctx, client)

	t := time.NewTicker(s.cfg.HeartbeatPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.sendHeartbeat(ctx, client)
		}
	}
}

func (s *Subsystem) sendHeartbeat(ctx context.Context, client *http.Client) {
	hb := heartbeatPayload{
		AgentID:   s.cfg.AgentID,
		UptimeS:   int64(time.Since(s.uptime).Seconds()),
		LastHash:  s.chain.Head(),
		WallClock: time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Best-effort log-file stat — failures don't abort the heartbeat;
	// they just leave the size/inode fields zero. The server-side
	// rule for "log_truncated" only fires when new < prev, so a
	// transient zero is recorded but doesn't spuriously alarm.
	if st, err := os.Stat(s.cfg.LogPath); err == nil {
		hb.LogFileSize = st.Size()
		hb.LogFileInode = statInode(st)
	}

	body, err := json.Marshal(hb)
	if err != nil {
		return
	}
	// Canonical X-Timestamp / X-Signature format — matches
	// /api/v1/agent/ingest. See VerifyHMAC in internal/api/middleware.go.
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, s.cfg.FleetSecret)
	mac.Write(body)
	mac.Write([]byte(ts))
	sig := hex.EncodeToString(mac.Sum(nil))

	url := strings.TrimRight(s.cfg.ServerURL, "/") + "/api/v1/agent/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", sig)
	req.Header.Set("X-Agent-ID", s.cfg.AgentID)

	resp, err := client.Do(req)
	if err != nil {
		s.log.Debug("heartbeat post failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		s.log.Debug("heartbeat returned HTTP %d", resp.StatusCode)
	}
}
