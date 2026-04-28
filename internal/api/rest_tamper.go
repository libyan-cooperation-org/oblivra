package api

// Tamper-evidence REST endpoints (Tamper Path 1, Layers 1 + 3 + 5).
//
// Layer 1 — agent operational log forwarding:
//   POST /api/v1/agent/oplog       agent → server batch of log lines
//   GET  /api/v1/agent/oplog       UI lookup (per-agent recent lines)
//
// Layer 3 — heartbeat:
//   POST /api/v1/agent/heartbeat   agent → server heartbeat
//
// Layer 5 — dashboard:
//   GET  /api/v1/integrity         per-agent integrity status snapshot
//
// Auth: oplog and heartbeat use the existing fleet-secret HMAC same
// as `/api/v1/agent/ingest`. They sit alongside that endpoint in the
// auth-bypass list and validate via VerifyHMAC inside the handler.
//
// The dashboard endpoint is operator-facing (analyst+).

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
)

// restoreBody wraps a byte slice in a fresh ReadCloser. Used by
// verifyAgentRequest after slurping the body for HMAC verification —
// the downstream JSON decoder still needs the same bytes.
func restoreBody(b []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(b))
}

// We import database/sql directly via the `sql` alias above; the
// internal/database package would create a redundant indirection
// since the tamper handlers only read/write the raw *sql.DB.

// ── Layer 1: oplog ───────────────────────────────────────────────

type oplogBatch struct {
	AgentID  string      `json:"agent_id"`
	BatchSeq int64       `json:"batch_seq"`
	Lines    []oplogLine `json:"lines"`
}
type oplogLine struct {
	TS       string `json:"ts"`
	Level    string `json:"level"`
	Source   string `json:"source"`
	Message  string `json:"message"`
	PrevHash string `json:"prev_hash"`
}

func (s *RESTServer) handleAgentOplog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleAgentOplogPOST(w, r)
	case http.MethodGet:
		s.handleAgentOplogGET(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *RESTServer) handleAgentOplogPOST(w http.ResponseWriter, r *http.Request) {
	// Agent auth — same HMAC fleet-secret as /api/v1/agent/ingest.
	if !s.verifyAgentRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var batch oplogBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if batch.AgentID == "" || len(batch.Lines) == 0 {
		http.Error(w, "agent_id and lines required", http.StatusBadRequest)
		return
	}
	db := s.databaseDB()
	if db == nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}

	// Idempotent insert via UNIQUE constraint. Use INSERT OR IGNORE
	// so a retry of an already-shipped batch is a no-op.
	stmt, err := db.Prepare(`INSERT OR IGNORE INTO agent_oplog
		(agent_id, batch_seq, ts, level, source, message, prev_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		http.Error(w, "prepare: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()
	inserted := 0
	for _, ln := range batch.Lines {
		if ln.TS == "" {
			ln.TS = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if ln.Level == "" {
			ln.Level = "INF"
		}
		res, err := stmt.Exec(batch.AgentID, batch.BatchSeq, ln.TS, ln.Level, ln.Source, ln.Message, ln.PrevHash)
		if err != nil {
			s.log.Warn("[oplog] insert agent=%s seq=%d: %v", batch.AgentID, batch.BatchSeq, err)
			continue
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"ok":       true,
		"received": len(batch.Lines),
		"inserted": inserted,
	})
}

func (s *RESTServer) handleAgentOplogGET(w http.ResponseWriter, r *http.Request) {
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin && role != auth.RoleReadOnly {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	agentID := r.URL.Query().Get("agent_id")
	limit := 200
	db := s.databaseDB()
	if db == nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}
	var rows *sql.Rows
	var err error
	if agentID != "" {
		rows, err = db.Query(`SELECT agent_id, batch_seq, ts, level, source, message, received_at
			FROM agent_oplog WHERE agent_id = ?
			ORDER BY received_at DESC LIMIT ?`, agentID, limit)
	} else {
		rows, err = db.Query(`SELECT agent_id, batch_seq, ts, level, source, message, received_at
			FROM agent_oplog ORDER BY received_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		http.Error(w, "query: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var (
			aid, ts, level, source, msg, received sql.NullString
			seq                                    sql.NullInt64
		)
		if err := rows.Scan(&aid, &seq, &ts, &level, &source, &msg, &received); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"agent_id":    aid.String,
			"batch_seq":   seq.Int64,
			"ts":          ts.String,
			"level":       level.String,
			"source":      source.String,
			"message":     msg.String,
			"received_at": received.String,
		})
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{"entries": out})
}

// ── Layer 3: heartbeat ───────────────────────────────────────────

type heartbeatPayload struct {
	AgentID      string  `json:"agent_id"`
	UptimeS      int64   `json:"uptime_s"`
	LogFileInode uint64  `json:"log_file_inode"`
	LogFileSize  int64   `json:"log_file_size"`
	LastHash     string  `json:"last_hash"`
	WallClock    string  `json:"wall_clock"`
}

func (s *RESTServer) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.verifyAgentRequest(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var hb heartbeatPayload
	if err := json.NewDecoder(r.Body).Decode(&hb); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if hb.AgentID == "" {
		http.Error(w, "agent_id required", http.StatusBadRequest)
		return
	}
	db := s.databaseDB()
	if db == nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}

	// Compute time skew vs server clock.
	skew := 0.0
	if t, err := time.Parse(time.RFC3339Nano, hb.WallClock); err == nil {
		skew = time.Since(t).Seconds()
	}

	// Fetch previous values so the tamper-rule scanner can compare.
	var prevSize, prevInode sql.NullInt64
	_ = db.QueryRow(`SELECT log_file_size, log_file_inode FROM agent_heartbeats WHERE agent_id = ?`,
		hb.AgentID).Scan(&prevSize, &prevInode)

	// Detection signals — emit alert events for tamper rules.
	if prevSize.Valid && hb.LogFileSize < prevSize.Int64 {
		s.publishTamperAlert(hb.AgentID, "log_truncated",
			fmt.Sprintf("log file shrank from %d to %d bytes", prevSize.Int64, hb.LogFileSize),
			"critical")
	}
	if prevInode.Valid && uint64(prevInode.Int64) != hb.LogFileInode && prevInode.Int64 != 0 {
		s.publishTamperAlert(hb.AgentID, "log_inode_changed",
			fmt.Sprintf("inode rotated from %d to %d", prevInode.Int64, hb.LogFileInode),
			"medium")
	}
	if skew > 60 || skew < -60 {
		s.publishTamperAlert(hb.AgentID, "time_skew",
			fmt.Sprintf("agent clock differs from server by %.1fs", skew),
			"info")
	}

	// Upsert: SQLite ON CONFLICT replaces on agent_id PK.
	_, err := db.Exec(`
		INSERT INTO agent_heartbeats
			(agent_id, received_at, wall_clock, uptime_s, log_file_inode, log_file_size, last_hash, prev_log_size, prev_log_inode, skew_seconds)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(agent_id) DO UPDATE SET
			received_at = CURRENT_TIMESTAMP,
			wall_clock = excluded.wall_clock,
			uptime_s = excluded.uptime_s,
			prev_log_inode = log_file_inode,
			prev_log_size  = log_file_size,
			log_file_inode = excluded.log_file_inode,
			log_file_size  = excluded.log_file_size,
			last_hash = excluded.last_hash,
			skew_seconds = excluded.skew_seconds`,
		hb.AgentID, hb.WallClock, hb.UptimeS, hb.LogFileInode, hb.LogFileSize, hb.LastHash,
		prevSize.Int64, prevInode.Int64, skew)
	if err != nil {
		http.Error(w, "upsert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{"ok": true})
}

// ── Layer 5: integrity dashboard ─────────────────────────────────

type integrityRow struct {
	AgentID        string  `json:"agent_id"`
	ReceivedAt     string  `json:"received_at"`
	UptimeS        int64   `json:"uptime_s"`
	LogFileSize    int64   `json:"log_file_size"`
	SkewSeconds    float64 `json:"skew_seconds"`
	SecondsSinceHB int64   `json:"seconds_since_heartbeat"`
	Status         string  `json:"status"` // OK | STALE | DARK | TAMPERED
	LastHash       string  `json:"last_hash,omitempty"`
}

func (s *RESTServer) handleIntegrity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin && role != auth.RoleReadOnly {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	db := s.databaseDB()
	if db == nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}
	rows, err := db.Query(`SELECT agent_id, received_at, uptime_s, log_file_size, skew_seconds, last_hash,
		(strftime('%s','now') - strftime('%s', received_at)) AS seconds_since
		FROM agent_heartbeats ORDER BY received_at DESC`)
	if err != nil {
		http.Error(w, "query: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := []integrityRow{}
	healthy, stale, dark, tampered := 0, 0, 0, 0
	for rows.Next() {
		var row integrityRow
		var lastHash sql.NullString
		var skew sql.NullFloat64
		if err := rows.Scan(&row.AgentID, &row.ReceivedAt, &row.UptimeS,
			&row.LogFileSize, &skew, &lastHash, &row.SecondsSinceHB); err != nil {
			continue
		}
		row.SkewSeconds = skew.Float64
		row.LastHash = lastHash.String
		row.Status = classifyIntegrity(row.SecondsSinceHB, row.SkewSeconds)
		switch row.Status {
		case "OK":
			healthy++
		case "STALE":
			stale++
		case "DARK":
			dark++
		case "TAMPERED":
			tampered++
		}
		out = append(out, row)
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"agents":         out,
		"healthy_count":  healthy,
		"stale_count":    stale,
		"dark_count":     dark,
		"tampered_count": tampered,
	})
}

// classifyIntegrity is the canonical OK / STALE / DARK / TAMPERED
// rubric. UI badge colour + Tactical Hub KPI tile both read this.
func classifyIntegrity(secondsSinceHB int64, skew float64) string {
	switch {
	case skew > 300 || skew < -300:
		// >5min skew is suspicious enough to flag tampered.
		return "TAMPERED"
	case secondsSinceHB > 7200:
		// 2h silent → assume off.
		return "DARK"
	case secondsSinceHB > 90:
		return "STALE"
	}
	return "OK"
}

// publishTamperAlert is a helper that fires through the existing
// alert pipeline. The alert shows up in the regular /alerts queue
// AND in the integrity dashboard via the audit-event subscriber.
func (s *RESTServer) publishTamperAlert(agentID, kind, detail, severity string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish("tamper:detected", map[string]any{
		"agent_id": agentID,
		"kind":     kind,
		"detail":   detail,
		"severity": severity,
		"ts":       time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// databaseDB pulls the *sql.DB handle from the s.db DatabaseStore
// interface. Centralised so individual handlers don't all repeat the
// `if s.db == nil` + `s.db.DB() == nil` dance.
func (s *RESTServer) databaseDB() *sql.DB {
	if s.db == nil {
		return nil
	}
	return s.db.DB()
}

// MissedHeartbeatScanner runs in the background and emits
// agent:heartbeat_missed for any agent whose latest heartbeat is
// older than 90s. Started by main.go alongside the other long-lived
// service loops.
func (s *RESTServer) StartHeartbeatScanner(ctx context.Context) {
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.scanMissedHeartbeats()
			}
		}
	}()
}

func (s *RESTServer) scanMissedHeartbeats() {
	db := s.databaseDB()
	if db == nil {
		return
	}
	rows, err := db.Query(`SELECT agent_id,
		(strftime('%s','now') - strftime('%s', received_at)) AS seconds_since
		FROM agent_heartbeats
		WHERE seconds_since > 90`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var aid string
		var since int64
		if err := rows.Scan(&aid, &since); err != nil {
			continue
		}
		// Don't spam the bus on every scan — only fire if the agent
		// just crossed the threshold (which we'd ideally track in a
		// separate "last_alarmed_at" column; for now the audit log
		// handles dedup via timestamp grouping).
		s.publishTamperAlert(aid, "heartbeat_missed",
			fmt.Sprintf("no heartbeat for %ds", since),
			heartbeatSeverity(since))
	}
}

func heartbeatSeverity(since int64) string {
	switch {
	case since > 7200:
		return "high" // 2h silent
	case since > 600:
		return "medium" // 10min silent
	}
	return "info"
}

// verifyAgentRequest validates an agent's HMAC-signed request by
// reading the body and delegating to the canonical VerifyHMAC helper
// (internal/api/middleware.go). Same headers + same MAC format as
// /api/v1/agent/ingest, so an agent that already authenticates against
// ingest authenticates here without changes.
//
// In addition to the HMAC check, this method consults the replay
// cache: a (agent_id, timestamp, body) fingerprint that's already
// been seen within the last 60s is rejected. That closes the window
// where an attacker who captured a valid request could re-POST it
// bit-for-bit and have it accepted (the HMAC validates either way).
//
// Returns false on any verification failure. The HTTP handler turns
// that into a 401. Body is replaced with a fresh ReadCloser so the
// downstream JSON decoder still works.
//
// Loopback exception: when no fleet secret is configured (fresh
// install / dev), allow loopback callers through with a one-time
// audit-log warning. Hard requirement when the secret IS configured.
func (s *RESTServer) verifyAgentRequest(r *http.Request) bool {
	if len(s.fleetSecret) == 0 {
		host := r.RemoteAddr
		if i := strings.LastIndex(host, ":"); i > 0 {
			host = host[:i]
		}
		if host == "127.0.0.1" || host == "::1" || host == "[::1]" {
			s.log.Warn("[tamper] accepting loopback agent request without HMAC — fleet_secret not configured")
			return true
		}
		return false
	}
	body, err := readAndRestoreBody(r)
	if err != nil {
		s.log.Debug("[tamper] body read: %v", err)
		return false
	}
	if err := VerifyHMAC(r, body, s.fleetSecret); err != nil {
		s.log.Debug("[tamper] HMAC reject: %v", err)
		return false
	}

	// Replay protection (audit fix #1). Fingerprint = (agent_id, ts, body).
	// Same agent posting an identical body at the identical timestamp
	// twice within 60s is either a network retry that we ALREADY
	// idempotent-handled at the DB layer, or a captured replay. We
	// reject it — DB idempotency is a defence-in-depth, not a primary.
	agentID := r.Header.Get("X-Agent-ID")
	ts := r.Header.Get("X-Timestamp")
	if s.replayCache != nil && s.replayCache.Seen(agentID, ts, body) {
		s.log.Warn("[tamper] replay rejected: agent=%s ts=%s body_bytes=%d",
			agentID, ts, len(body))
		return false
	}
	return true
}

// readAndRestoreBody slurps the request body and replaces r.Body so
// the downstream JSON decoder still has bytes to read. Reused by
// verifyAgentRequest because VerifyHMAC needs the raw bytes for the
// MAC computation.
func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return nil, err
	}
	r.Body = restoreBody(body)
	return body, nil
}
