package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/parquet-go/parquet-go"
)

// ParquetLogEntry represents the schema for archived terminal logs
type ParquetLogEntry struct {
	Timestamp int64  `parquet:"timestamp"` // Unix microseconds
	SessionID string `parquet:"session_id,dict"`
	Host      string `parquet:"host,dict"`
	Output    string `parquet:"output"`
}

// ParquetArchiver handles native writing and reading of Parquet files
type ParquetArchiver struct {
	dataDir string
	log     *logger.Logger
}

// NewParquetArchiver initializes the archiver
func NewParquetArchiver(dataDir string, log *logger.Logger) *ParquetArchiver {
	return &ParquetArchiver{
		dataDir: dataDir,
		log:     log,
	}
}

// WriteBatch writes a batch of log entries to a daily partitioned Parquet file.
// Path: <dataDir>/archives/yyyy/mm/dd/logs.parquet
func (pa *ParquetArchiver) WriteBatch(date time.Time, entries []ParquetLogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Generate partition path: dataDir/archives/2026/03/01/logs.parquet
	year := fmt.Sprintf("%04d", date.Year())
	month := fmt.Sprintf("%02d", date.Month())
	day := fmt.Sprintf("%02d", date.Day())

	dirPath := filepath.Join(pa.dataDir, "archives", year, month, day)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	filePath := filepath.Join(dirPath, "logs.parquet")

	// Open file for appending or create new
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open parquet file: %w", err)
	}
	defer f.Close()

	// If file exists and has size, we need to handle append.
	// However, parquet-go doesn't support true appending to existing files seamlessly
	// without reading the schema/footer first and rewriting or using a RowGroup writer.
	// For simplicity in this demo phase, if it's a new day, we just create the file.
	// If it already exists, we will create a timestamped file to avoid complex merging.
	info, err := f.Stat()
	if err == nil && info.Size() > 0 {
		f.Close()
		timestampedPath := filepath.Join(dirPath, fmt.Sprintf("logs_%d.parquet", time.Now().Unix()))
		f, err = os.OpenFile(timestampedPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("create timestamped parquet file: %w", err)
		}
		filePath = timestampedPath
		defer f.Close()
	}

	// Create a writer with ZSTD compression (excellent ratio for text logs)
	writer := parquet.NewGenericWriter[ParquetLogEntry](f,
		parquet.Compression(&parquet.Zstd),
	)

	// Write all rows
	_, err = writer.Write(entries)
	if err != nil {
		writer.Close()
		return fmt.Errorf("write parquet rows: %w", err)
	}

	// Close the writer to flush the footer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close parquet writer: %w", err)
	}

	if pa.log != nil {
		pa.log.Info("[ARCHIVER] Wrote %d events to %s", len(entries), filePath)
	}

	return nil
}

// ReadPartition reads all entries from a specific daily partition
func (pa *ParquetArchiver) ReadPartition(date time.Time) ([]ParquetLogEntry, error) {
	year := fmt.Sprintf("%04d", date.Year())
	month := fmt.Sprintf("%02d", date.Month())
	day := fmt.Sprintf("%02d", date.Day())

	dirPath := filepath.Join(pa.dataDir, "archives", year, month, day)

	var allEntries []ParquetLogEntry

	// Find all parquet files in this directory
	files, err := filepath.Glob(filepath.Join(dirPath, "*.parquet"))
	if err != nil || len(files) == 0 {
		return nil, nil // No archives for this day
	}

	for _, filePath := range files {
		f, err := os.Open(filePath)
		if err != nil {
			pa.log.Error("Failed to open archive %s: %v", filePath, err)
			continue
		}

		reader := parquet.NewGenericReader[ParquetLogEntry](f)

		// Read rows in chunks to avoid massive memory spikes
		chunk := make([]ParquetLogEntry, 1000)
		for {
			n, err := reader.Read(chunk)
			if n > 0 {
				baseEntries := make([]ParquetLogEntry, n)
				copy(baseEntries, chunk[:n])
				allEntries = append(allEntries, baseEntries...)
			}
			if err != nil {
				break // EOF or actual error, loop terminates
			}
		}

		f.Close()
	}

	return allEntries, nil
}
