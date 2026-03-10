package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/parquet-go/parquet-go"
)

// Archiver handles native out-of-process Parquet archiving
type Archiver struct {
	db          *sql.DB
	archivePath string
	retention   time.Duration
	log         *logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// ParquetFrame represents the schema for archived recording frames
type ParquetFrame struct {
	RecordingID string  `parquet:"recording_id,dict"`
	Timestamp   float64 `parquet:"timestamp"`
	Type        string  `parquet:"type,dict"`
	Data        string  `parquet:"data"`
}

// NewArchiver ensures the required paths and configurations are set
func NewArchiver(db *sql.DB, baseDir string, retention time.Duration, log *logger.Logger) *Archiver {
	ctx, cancel := context.WithCancel(context.Background())
	return &Archiver{
		db:          db,
		archivePath: filepath.Join(baseDir, "archives"),
		retention:   retention,
		log:         log,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the periodic archiving loop
func (a *Archiver) Start() {
	a.wg.Add(1)
	defer a.wg.Done()

	if err := os.MkdirAll(a.archivePath, 0700); err != nil {
		if a.log != nil {
			a.log.Error("[ARCHIVER] Failed to create archive directory: %v", err)
		}
		return
	}

	ticker := time.NewTicker(1 * time.Hour) // Run maintenance hourly
	defer ticker.Stop()

	if a.log != nil {
		a.log.Info("[ARCHIVER] Started background archiver. Retention: %v", a.retention)
	}

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.performArchive(); err != nil && a.log != nil {
				a.log.Error("[ARCHIVER] Maintenance cycle failed: %v", err)
			}
		}
	}
}

// Stop gracefully shuts down the archiver
func (a *Archiver) Stop() {
	a.cancel()
	a.wg.Wait()
}

// performArchive extracts old records, writes them natively to a ZSTD compressed Parquet file, and then prunes SQLite
func (a *Archiver) performArchive() error {
	cutoff := time.Now().Add(-a.retention).Unix()

	// 1. Check if there are any records to archive
	var count int
	err := a.db.QueryRow(`SELECT count(*) FROM recording_frames WHERE timestamp < ?`, cutoff).Scan(&count)
	if err != nil {
		return fmt.Errorf("count query failed: %w", err)
	}

	if count == 0 {
		return nil // Nothing to archive
	}

	if a.log != nil {
		a.log.Info("[ARCHIVER] Found %d recording frames older than cutoff. Starting extraction...", count)
	}

	// 2. Query the rows
	rows, err := a.db.Query(`SELECT recording_id, timestamp, type, data FROM recording_frames WHERE timestamp < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("extraction query failed: %w", err)
	}
	defer rows.Close()

	// 3. Setup Parquet Writer
	parquetFilename := fmt.Sprintf("archive_%s.parquet", time.Now().Format("2006-01-02_15-04-05"))
	parquetFilePath := filepath.Join(a.archivePath, parquetFilename)

	f, err := os.Create(parquetFilePath)
	if err != nil {
		return fmt.Errorf("failed to create parquet file: %w", err)
	}
	defer f.Close()

	writer := parquet.NewGenericWriter[ParquetFrame](f, parquet.Compression(&parquet.Zstd))

	// 4. Extract and Write
	var recID, typ, data string
	var ts float64
	var batch []ParquetFrame

	for rows.Next() {
		if err := rows.Scan(&recID, &ts, &typ, &data); err != nil {
			if a.log != nil {
				a.log.Warn("[ARCHIVER] Skipping row during extract: %v", err)
			}
			continue
		}

		batch = append(batch, ParquetFrame{
			RecordingID: recID,
			Timestamp:   ts,
			Type:        typ,
			Data:        data,
		})

		// Flush in chunks of 5000 to avoid high memory spikes
		if len(batch) >= 5000 {
			if _, err := writer.Write(batch); err != nil {
				return fmt.Errorf("write parquet chunk: %w", err)
			}
			batch = batch[:0]
		}
	}

	// Flush remaining
	if len(batch) > 0 {
		if _, err := writer.Write(batch); err != nil {
			return fmt.Errorf("write final parquet chunk: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close parquet writer: %w", err)
	}

	if a.log != nil {
		a.log.Info("[ARCHIVER] Successfully created native Parquet archive: %s", parquetFilePath)
	}

	// 5. Prune SQLite
	res, err := a.db.Exec(`DELETE FROM recording_frames WHERE timestamp < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("failed to prune SQLite after archiving: %w", err)
	}

	deletedCount, _ := res.RowsAffected()
	if a.log != nil {
		a.log.Info("[ARCHIVER] SQLite Pruned. Removed %d rows.", deletedCount)
	}

	return nil
}
