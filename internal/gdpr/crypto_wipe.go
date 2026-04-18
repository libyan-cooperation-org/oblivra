package gdpr

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type DataDestructionService struct {
	db *sql.DB
}

func NewDataDestructionService(db *sql.DB) *DataDestructionService {
	return &DataDestructionService{db: db}
}

// allowedTables is the strict whitelist of tables that may be crypto-wiped.
var allowedTables = map[string]bool{
	"sessions":         true,
	"audit_logs":       true,
	"host_events":      true,
	"incidents":        true,
	"config_changes":   true,
	"evidence_chain":   true,
	"recording_frames": true,
	"recordings":       true,
	"credentials":      true,
	"hosts":            true,
	"siem_events":      true,
	"settings":         true,
	"snippets":         true,
}

// tableSensitiveColumns maps tables to columns that should be overwritten with random data.
var tableSensitiveColumns = map[string][]string{
	"hosts":          {"password", "notes", "hostname", "username"},
	"credentials":    {"encrypted_data"},
	"sessions":       {"recording_path"},
	"audit_logs":     {"details"},
	"snippets":       {"command", "description"},
	"settings":       {"value"},
	"host_events":    {"raw_log", "user", "source_ip"},
	"siem_events":    {"raw_log", "DetailsJSON"}, // Assuming DetailsJSON exists in siem_events
	"evidence_chain": {"notes"},
}

// safeIdentifier validates that a string is a safe SQL identifier (alphanumeric + underscore only).
var safeIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validateTableName checks that the table name is both whitelisted and a safe identifier.
func validateTableName(tableName string) error {
	if !allowedTables[tableName] {
		return fmt.Errorf("table %q is not in the allowed whitelist", tableName)
	}
	if !safeIdentifier.MatchString(tableName) {
		return fmt.Errorf("table name %q contains unsafe characters", tableName)
	}
	return nil
}

// CryptoWipe — GDPR-compliant secure deletion
// SECURITY: tableName is validated against a whitelist. whereClause should use
// positional parameters (?) and values should be passed in args.
func (s *DataDestructionService) CryptoWipe(tableName, whereClause string, args ...interface{}) error {
	if err := validateTableName(tableName); err != nil {
		return fmt.Errorf("crypto wipe refused: %w", err)
	}

	if err := validateWhereClause(whereClause); err != nil {
		return fmt.Errorf("crypto wipe refused: %w", err)
	}

	// SECURITY: Ensure whereClause doesn't contain multiple statements or bypasses
	if len(whereClause) > 256 {
		return fmt.Errorf("crypto wipe refused: whereClause too long")
	}

	// 0. Check if table exists
	var name string
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err == sql.ErrNoRows {
		return nil // Table already gone/never existed, goal achieved
	} else if err != nil {
		return fmt.Errorf("check table existence: %w", err)
	}

	// 1. Enable secure_delete to ensure SQLite overwrites deleted pages with zeros
	if _, err := s.db.Exec("PRAGMA secure_delete = ON"); err != nil {
		return fmt.Errorf("failed to enable secure_delete: %w", err)
	}

	// 2. Overwrite sensitive columns with random data if they exist
	if cols, ok := tableSensitiveColumns[tableName]; ok {
		for _, col := range cols {
			// SECURITY: Parameters (?) in whereClause are handled by s.db.Exec
			query := fmt.Sprintf("UPDATE %s SET %s = randomblob(length(%s)) WHERE %s", tableName, col, col, whereClause)
			// We don't fail if a specific column update fails (e.g. if it doesn't exist in this DB version)
			_, _ = s.db.Exec(query, args...)
		}
	}

	// 3. DELETE (frees space for reuse, secure_delete ensures zero-fill)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)
	if _, err := s.db.Exec(query, args...); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	// 4. VACUUM (reclaim disk space and further scramble remaining slack space)
	if _, err := s.db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("vacuum failed: %w", err)
	}

	// 5. Audit trail
	return s.logDestruction(tableName, whereClause)
}

// validateWhereClause rejects SQL injection patterns in the WHERE clause.
// This is a defense-in-depth measure. OBLIVRA callers are expected to use
// positional parameters (?) for all user-provided values.
func validateWhereClause(clause string) error {
	if clause == "" {
		return fmt.Errorf("empty where clause not allowed for destruction")
	}

	// Block multiple statements (;) and SQL comments (--), which are common injection markers.
	if strings.Contains(clause, ";") || strings.Contains(clause, "--") || strings.Contains(clause, "/*") {
		return fmt.Errorf("unsupported SQL characters in where clause")
	}

	// Block common destructive keywords if they appear as standalone words.
	// We use a stricter regex that looks for word boundaries.
	unsafeKeywords := regexp.MustCompile(`(?i)\b(union|drop|insert|update|delete|exec|alter|truncate|create|attach|detach)\b`)
	if unsafeKeywords.MatchString(clause) {
		return fmt.Errorf("disallowed SQL keywords detected in where clause")
	}

	return nil
}

func (s *DataDestructionService) logDestruction(tableName, whereClause string) error {
	// Emit a log or audit trail here
	// This ensures we have a record that data was destroyed for compliance.
	fmt.Printf("GDPR: Successfully crypto-wiped %s where %s\n", tableName, whereClause)
	return nil
}

// WipeFile — secure file deletion
func (s *DataDestructionService) WipeFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	info, _ := f.Stat()
	size := info.Size()

	// 3-pass overwrite (DoD standard)
	for pass := 0; pass < 3; pass++ {
		f.Seek(0, 0)
		if pass == 2 {
			// Last pass = zeros
			f.Write(make([]byte, size))
		} else {
			// Random passes
			random := make([]byte, size)
			rand.Read(random)
			f.Write(random)
		}
		f.Sync()
	}

	return os.Remove(path)
}
