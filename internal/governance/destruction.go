package governance

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DestructionRecord is an immutable audit entry for data wipe operations.
type DestructionRecord struct {
	ID          string    `json:"id"`
	Timestamp   string    `json:"timestamp"`
	InitiatedBy string    `json:"initiated_by"`
	Scope       string    `json:"scope"`  // "credential", "host_events", "full_database"
	Method      string    `json:"method"` // "crypto_wipe", "sql_delete"
	ItemCount   int       `json:"item_count"`
	Status      string    `json:"status"` // "completed", "failed"
}

// DataDestructionService handles secure, auditable data erasure.
type DataDestructionService struct {
	bus     *eventbus.Bus
	log     *logger.Logger
	history []DestructionRecord
}

// NewDataDestructionService creates a new data destruction handler.
func NewDataDestructionService(bus *eventbus.Bus, log *logger.Logger) *DataDestructionService {
	return &DataDestructionService{
		bus: bus,
		log: log.WithPrefix("data_destruction"),
	}
}

// CryptoWipeFile overwrites a file with cryptographically random bytes before deleting it.
// This is a NIST SP 800-88 "Clear" level purge (single overwrite with random data).
func (s *DataDestructionService) CryptoWipeFile(path string, initiatedBy string) error {
	s.log.Warn("[WIPE] Crypto-wiping file: %s (by: %s)", path, initiatedBy)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat target: %w", err)
	}

	size := info.Size()

	// Overwrite with random data
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open for wipe: %w", err)
	}

	randomBuf := make([]byte, 4096)
	written := int64(0)
	for written < size {
		n := int64(len(randomBuf))
		if size-written < n {
			n = size - written
		}
		rand.Read(randomBuf[:n])
		w, err := f.Write(randomBuf[:n])
		if err != nil {
			f.Close()
			return fmt.Errorf("overwrite failed at offset %d: %w", written, err)
		}
		written += int64(w)
	}
	f.Sync()
	f.Close()

	// Delete after overwrite
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove after wipe: %w", err)
	}

	record := DestructionRecord{
		ID:          fmt.Sprintf("wipe_%d", time.Now().UnixNano()),
		Timestamp:   time.Now().Format(time.RFC3339),
		InitiatedBy: initiatedBy,
		Scope:       "file:" + path,
		Method:      "crypto_wipe",
		ItemCount:   1,
		Status:      "completed",
	}
	s.history = append(s.history, record)

	if s.bus != nil {
		s.bus.Publish("governance.data_destroyed", record)
	}

	s.log.Info("[WIPE] File crypto-wiped successfully: %s (%d bytes)", path, size)
	return nil
}

// GetHistory returns the full audit trail of destruction operations.
func (s *DataDestructionService) GetHistory() []DestructionRecord {
	return s.history
}
