// Package lookup provides an in-memory lookup table engine for the OBLIVRA
// enrichment pipeline. Tables support CSV/JSON ingestion and Exact, CIDR,
// Wildcard, and Regex match strategies — fulfilling Phase 1.3 requirements.
package lookup

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// MatchType selects how a lookup value is compared during a query.
type MatchType string

const (
	MatchExact    MatchType = "exact"
	MatchCIDR     MatchType = "cidr"
	MatchWildcard MatchType = "wildcard"
	MatchRegex    MatchType = "regex"
)

// Table is a named lookup with a list of key→metadata rows.
type Table struct {
	Name      string            `json:"name"`
	MatchType MatchType         `json:"match_type"`
	Fields    []string          `json:"fields"` // Column names (first = key)
	Rows      []map[string]string `json:"rows"`

	// Pre-compiled index for fast lookups
	exactIdx  map[string]map[string]string
	cidrNets  []*net.IPNet
	cidrRows  []map[string]string
	wildcards []wildcardEntry
	regexes   []regexEntry
	mu        sync.RWMutex
}

type wildcardEntry struct {
	pattern string
	row     map[string]string
}

type regexEntry struct {
	re  *regexp.Regexp
	row map[string]string
}

// Manager holds all registered lookup tables and exposes CRUD + query operations.
type Manager struct {
	mu     sync.RWMutex
	tables map[string]*Table
}

func NewManager() *Manager {
	m := &Manager{tables: make(map[string]*Table)}

	// Seed built-in lookups
	m.seedBuiltins()
	return m
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

// List returns metadata for all registered tables (rows omitted for brevity).
func (m *Manager) List() []Table {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Table, 0, len(m.tables))
	for _, t := range m.tables {
		out = append(out, Table{Name: t.Name, MatchType: t.MatchType, Fields: t.Fields})
	}
	return out
}

// Get returns a full table including its rows.
func (m *Manager) Get(name string) (*Table, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tables[name]
	return t, ok
}

// UpsertFromCSV creates or replaces a table from CSV data.
func (m *Manager) UpsertFromCSV(name string, mt MatchType, data io.Reader) error {
	reader := csv.NewReader(data)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("csv parse: %w", err)
	}
	if len(records) < 1 {
		return fmt.Errorf("csv is empty")
	}

	fields := records[0]
	rows := make([]map[string]string, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := make(map[string]string, len(fields))
		for i, f := range fields {
			if i < len(rec) {
				row[f] = rec[i]
			}
		}
		rows = append(rows, row)
	}

	t := &Table{Name: name, MatchType: mt, Fields: fields, Rows: rows}
	t.buildIndex()

	m.mu.Lock()
	m.tables[name] = t
	m.mu.Unlock()
	return nil
}

// UpsertFromJSON creates or replaces a table from JSON rows.
func (m *Manager) UpsertFromJSON(name string, mt MatchType, data io.Reader) error {
	var payload struct {
		Fields []string            `json:"fields"`
		Rows   []map[string]string `json:"rows"`
	}
	if err := json.NewDecoder(data).Decode(&payload); err != nil {
		return fmt.Errorf("json parse: %w", err)
	}

	t := &Table{Name: name, MatchType: mt, Fields: payload.Fields, Rows: payload.Rows}
	t.buildIndex()

	m.mu.Lock()
	m.tables[name] = t
	m.mu.Unlock()
	return nil
}

// Delete removes a table by name.
func (m *Manager) Delete(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tables[name]; !ok {
		return false
	}
	delete(m.tables, name)
	return true
}

// ── Query ─────────────────────────────────────────────────────────────────────

// Lookup queries a table for a given key. Returns nil if no match is found.
func (m *Manager) Lookup(tableName, key string) map[string]string {
	m.mu.RLock()
	t, ok := m.tables[tableName]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	return t.lookup(key)
}

func (t *Table) lookup(key string) map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	switch t.MatchType {
	case MatchExact:
		if row, ok := t.exactIdx[strings.ToLower(key)]; ok {
			return row
		}
	case MatchCIDR:
		ip := net.ParseIP(key)
		if ip != nil {
			for i, net := range t.cidrNets {
				if net.Contains(ip) {
					return t.cidrRows[i]
				}
			}
		}
	case MatchWildcard:
		for _, w := range t.wildcards {
			matched, _ := filepath.Match(w.pattern, key)
			if matched {
				return w.row
			}
		}
	case MatchRegex:
		for _, r := range t.regexes {
			if r.re.MatchString(key) {
				return r.row
			}
		}
	}
	return nil
}

// ── Indexing ──────────────────────────────────────────────────────────────────

func (t *Table) buildIndex() {
	t.mu.Lock()
	defer t.mu.Unlock()

	keyField := ""
	if len(t.Fields) > 0 {
		keyField = t.Fields[0]
	}

	t.exactIdx  = make(map[string]map[string]string, len(t.Rows))
	t.cidrNets  = nil
	t.cidrRows  = nil
	t.wildcards = nil
	t.regexes   = nil

	for _, row := range t.Rows {
		key := row[keyField]
		switch t.MatchType {
		case MatchExact:
			t.exactIdx[strings.ToLower(key)] = row
		case MatchCIDR:
			_, ipNet, err := net.ParseCIDR(key)
			if err == nil {
				t.cidrNets = append(t.cidrNets, ipNet)
				t.cidrRows = append(t.cidrRows, row)
			}
		case MatchWildcard:
			t.wildcards = append(t.wildcards, wildcardEntry{pattern: key, row: row})
		case MatchRegex:
			re, err := regexp.Compile(key)
			if err == nil {
				t.regexes = append(t.regexes, regexEntry{re: re, row: row})
			}
		}
	}
}

// ── Built-in lookups ──────────────────────────────────────────────────────────

func (m *Manager) seedBuiltins() {
	// RFC 1918 private address ranges
	m.UpsertFromCSV("rfc1918_private", MatchCIDR, strings.NewReader(
		"cidr,label,description\n"+
			"10.0.0.0/8,Private-A,Class A private network\n"+
			"172.16.0.0/12,Private-B,Class B private network\n"+
			"192.168.0.0/16,Private-C,Class C private network\n"+
			"127.0.0.0/8,Loopback,Loopback addresses\n"+
			"169.254.0.0/16,LinkLocal,APIPA link-local addresses\n",
	))

	// Common port-to-service mapping
	m.UpsertFromCSV("port_service", MatchExact, strings.NewReader(
		"port,service,protocol,risk\n"+
			"22,SSH,TCP,medium\n"+
			"23,Telnet,TCP,high\n"+
			"25,SMTP,TCP,medium\n"+
			"53,DNS,UDP,low\n"+
			"80,HTTP,TCP,low\n"+
			"443,HTTPS,TCP,low\n"+
			"445,SMB,TCP,high\n"+
			"3389,RDP,TCP,high\n"+
			"5432,PostgreSQL,TCP,medium\n"+
			"3306,MySQL,TCP,medium\n"+
			"6379,Redis,TCP,high\n"+
			"27017,MongoDB,TCP,high\n",
	))

	// MITRE technique name lookup
	m.UpsertFromCSV("mitre_techniques", MatchExact, strings.NewReader(
		"technique_id,name,tactic\n"+
			"T1059,Command and Scripting Interpreter,Execution\n"+
			"T1078,Valid Accounts,Defense Evasion\n"+
			"T1110,Brute Force,Credential Access\n"+
			"T1021,Remote Services,Lateral Movement\n"+
			"T1055,Process Injection,Privilege Escalation\n"+
			"T1486,Data Encrypted for Impact,Impact\n"+
			"T1040,Network Sniffing,Credential Access\n"+
			"T1048,Exfiltration Over Alternative Protocol,Exfiltration\n",
	))
}
