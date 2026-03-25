package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// CommandHistoryService stores and retrieves per-host command history.
// Commands are stored in SQLite via the SIEM store's host event mechanism,
// and are surfaced in the terminal autocomplete overlay.
type CommandHistoryService struct {
	BaseService
	db       database.DatabaseStore
	log      *logger.Logger
	mu       sync.RWMutex
	// In-memory recent cache: hostID → recent commands (ring buffer, last 100)
	cache    map[string]*commandRing
}

// CommandEntry represents a single command in the history.
type CommandEntry struct {
	Command   string `json:"command"`
	HostID    string `json:"host_id"`
	Timestamp string `json:"timestamp"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

// commandRing is a simple ring buffer for recent commands per host.
type commandRing struct {
	entries []CommandEntry
	maxSize int
}

func newCommandRing(maxSize int) *commandRing {
	return &commandRing{
		entries: make([]CommandEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

func (r *commandRing) Add(entry CommandEntry) {
	if len(r.entries) >= r.maxSize {
		r.entries = r.entries[1:] // Drop oldest
	}
	r.entries = append(r.entries, entry)
}

func (r *commandRing) Search(prefix string) []string {
	var results []string
	seen := make(map[string]bool)
	// Search backwards (most recent first)
	for i := len(r.entries) - 1; i >= 0; i-- {
		cmd := r.entries[i].Command
		if strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(prefix)) && !seen[cmd] {
			results = append(results, cmd)
			seen[cmd] = true
		}
		if len(results) >= 20 {
			break
		}
	}
	return results
}

func (r *commandRing) GetRecent(limit int) []string {
	var results []string
	seen := make(map[string]bool)
	for i := len(r.entries) - 1; i >= 0; i-- {
		cmd := r.entries[i].Command
		if !seen[cmd] {
			results = append(results, cmd)
			seen[cmd] = true
		}
		if len(results) >= limit {
			break
		}
	}
	return results
}

func (s *CommandHistoryService) Name() string        { return "command-history" }
func (s *CommandHistoryService) Dependencies() []string { return []string{} }
func (s *CommandHistoryService) Start(ctx context.Context) error { return nil }
func (s *CommandHistoryService) Stop(ctx context.Context) error  { return nil }

// NewCommandHistoryService creates the per-host command history service.
func NewCommandHistoryService(db database.DatabaseStore, log *logger.Logger) *CommandHistoryService {
	svc := &CommandHistoryService{
		db:    db,
		log:   log.WithPrefix("cmd-history"),
		cache: make(map[string]*commandRing),
	}

	// Ensure the command_history table exists
	svc.ensureTable()

	return svc
}

// RecordCommand stores a command for a specific host.
func (s *CommandHistoryService) RecordCommand(hostID string, command string) error {
	command = strings.TrimSpace(command)
	if command == "" || len(command) < 2 {
		return nil // Skip empty or trivially short commands
	}

	// Skip obvious non-commands (enter, control sequences)
	if strings.HasPrefix(command, "\x1b") || command == "\r" || command == "\n" {
		return nil
	}

	entry := CommandEntry{
		Command:   command,
		HostID:    hostID,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add to in-memory cache
	s.mu.Lock()
	ring, ok := s.cache[hostID]
	if !ok {
		ring = newCommandRing(500) // Keep last 500 per host in memory
		s.cache[hostID] = ring
	}
	ring.Add(entry)
	s.mu.Unlock()

	// Persist to SQLite asynchronously
	go s.persistCommand(entry)

	return nil
}

// GetHistory returns recent commands for a specific host.
func (s *CommandHistoryService) GetHistory(hostID string, limit int) []string {
	if limit <= 0 {
		limit = 50
	}

	s.mu.RLock()
	ring, ok := s.cache[hostID]
	s.mu.RUnlock()

	if ok {
		return ring.GetRecent(limit)
	}

	// Fall back to SQLite
	return s.loadFromDB(hostID, limit)
}

// SearchHistory searches command history for a host with a prefix.
func (s *CommandHistoryService) SearchHistory(hostID string, prefix string) []string {
	s.mu.RLock()
	ring, ok := s.cache[hostID]
	s.mu.RUnlock()

	if ok {
		return ring.Search(prefix)
	}

	return nil
}

// GetGlobalHistory returns recent commands across all hosts (for fuzzy search).
func (s *CommandHistoryService) GetGlobalHistory(limit int) []CommandEntry {
	if limit <= 0 {
		limit = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []CommandEntry
	for _, ring := range s.cache {
		all = append(all, ring.entries...)
	}

	// Sort by timestamp descending
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all
}

// ──────────────────────────────────────────────
// SQLite persistence
// ──────────────────────────────────────────────

func (s *CommandHistoryService) ensureTable() {
	conn, err := s.db.Conn()
	if err != nil {
		s.log.Error("[CMD-HISTORY] Failed to get DB connection: %v", err)
		return
	}

	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id TEXT NOT NULL,
			command TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			exit_code INTEGER DEFAULT 0,
			UNIQUE(host_id, command, timestamp)
		)
	`)
	if err != nil {
		s.log.Error("[CMD-HISTORY] Failed to create table: %v", err)
	}

	// Index for fast per-host lookup
	_, _ = conn.Exec(`CREATE INDEX IF NOT EXISTS idx_cmd_history_host ON command_history(host_id, timestamp DESC)`)
}

func (s *CommandHistoryService) persistCommand(entry CommandEntry) {
	conn, err := s.db.Conn()
	if err != nil {
		return
	}

	_, err = conn.Exec(`
		INSERT OR IGNORE INTO command_history (host_id, command, timestamp, exit_code)
		VALUES (?, ?, ?, ?)
	`, entry.HostID, entry.Command, entry.Timestamp, entry.ExitCode)
	if err != nil {
		s.log.Debug("[CMD-HISTORY] Persist failed: %v", err)
	}

	// Prune old entries (keep last 500 per host)
	_, _ = conn.Exec(`
		DELETE FROM command_history WHERE host_id = ? AND id NOT IN (
			SELECT id FROM command_history WHERE host_id = ? ORDER BY timestamp DESC LIMIT 500
		)
	`, entry.HostID, entry.HostID)
}

func (s *CommandHistoryService) loadFromDB(hostID string, limit int) []string {
	conn, err := s.db.Conn()
	if err != nil {
		return nil
	}

	rows, err := conn.Query(`
		SELECT DISTINCT command FROM command_history
		WHERE host_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, hostID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var commands []string
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err == nil {
			commands = append(commands, cmd)
		}
	}

	// Populate cache from DB load
	s.mu.Lock()
	ring := newCommandRing(500)
	for i := len(commands) - 1; i >= 0; i-- {
		ring.Add(CommandEntry{
			Command: commands[i],
			HostID:  hostID,
		})
	}
	s.cache[hostID] = ring
	s.mu.Unlock()

	return commands
}

// LoadHostCache pre-loads command history for a specific host into memory.
func (s *CommandHistoryService) LoadHostCache(hostID string) {
	_ = s.loadFromDB(hostID, 500)
	s.log.Debug("[CMD-HISTORY] Cache loaded for host %s", hostID)
}

// GetSuggestions returns command suggestions combining history + prefix matching.
// This is the primary API used by the frontend autocomplete overlay.
func (s *CommandHistoryService) GetSuggestions(hostID string, input string) []string {
	if strings.TrimSpace(input) == "" {
		return s.GetHistory(hostID, 10)
	}

	results := s.SearchHistory(hostID, input)

	// Add common shell completions for common prefixes
	commonCompletions := getCommonCompletions(input)
	seen := make(map[string]bool)
	for _, r := range results {
		seen[r] = true
	}
	for _, c := range commonCompletions {
		if !seen[c] {
			results = append(results, c)
		}
	}

	if len(results) > 20 {
		results = results[:20]
	}
	return results
}

// getCommonCompletions returns common shell command completions for a prefix.
func getCommonCompletions(prefix string) []string {
	common := []string{
		"ls -la", "ls -lah", "ps aux", "ps -ef",
		"top", "htop", "df -h", "du -sh *",
		"grep -r", "grep -rn", "find . -name",
		"tail -f", "tail -n 100",
		"cat", "less", "head",
		"systemctl status", "systemctl restart", "journalctl -u",
		"docker ps", "docker logs", "docker exec -it",
		"kubectl get pods", "kubectl logs",
		"netstat -tlnp", "ss -tlnp",
		"iptables -L -n", "ip addr",
		"curl -v", "wget",
		"chmod", "chown", "mkdir -p",
		"tar -xzf", "tar -czf",
		"vim", "nano", "vi",
		"git status", "git log --oneline",
	}

	prefix = strings.ToLower(prefix)
	var matches []string
	for _, cmd := range common {
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// ClearHistory removes all command history for a specific host.
func (s *CommandHistoryService) ClearHistory(hostID string) error {
	s.mu.Lock()
	delete(s.cache, hostID)
	s.mu.Unlock()

	conn, err := s.db.Conn()
	if err != nil {
		return fmt.Errorf("get db conn: %w", err)
	}

	_, err = conn.Exec("DELETE FROM command_history WHERE host_id = ?", hostID)
	return err
}
