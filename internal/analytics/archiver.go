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

// ParquetLogEntry represents the schema for archived terminal logs
type ParquetLogEntry struct {
	Timestamp string `parquet:"timestamp,dict"`
	TenantID  string `parquet:"tenant_id,dict"`
	SessionID string `parquet:"session_id,dict"`
	Host      string `parquet:"host,dict"`
	Output    string `parquet:"output"`
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
		wg:          sync.WaitGroup{},
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

	// Run once on startup
	if err := a.performArchive(); err != nil && a.log != nil {
		a.log.Error("[ARCHIVER] Initial maintenance cycle failed: %v", err)
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

// performArchive orchestrates the archiving of multiple telemetry tables
func (a *Archiver) performArchive() error {
	// 1. Archive Recording Frames
	if err := a.archiveRecordingFrames(); err != nil {
		return err
	}

	// 2. Archive Terminal Logs
	if err := a.archiveTerminalLogs(); err != nil {
		return err
	}

	return nil
}

func (a *Archiver) archiveRecordingFrames() error {
	cutoff := time.Now().Add(-a.retention).Unix()
	
	rows, err := a.db.Query(`SELECT recording_id, timestamp, type, data FROM recording_frames WHERE timestamp < ?`, cutoff)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	parquetFilename := fmt.Sprintf("recordings_%s.parquet", time.Now().Format("2006-01-02_15-04-05"))
	parquetFilePath := filepath.Join(a.archivePath, parquetFilename)

	f, err := os.Create(parquetFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := parquet.NewGenericWriter[ParquetFrame](f, parquet.Compression(&parquet.Zstd))
	var batch []ParquetFrame

	for rows.Next() {
		var recID, typ, data string
		var ts float64
		if err := rows.Scan(&recID, &ts, &typ, &data); err != nil {
			continue
		}
		batch = append(batch, ParquetFrame{recID, ts, typ, data})
		count++
		if len(batch) >= 5000 {
			writer.Write(batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		writer.Write(batch)
	}
	writer.Close()

	if count > 0 {
		a.db.Exec(`DELETE FROM recording_frames WHERE timestamp < ?`, cutoff)
		if a.log != nil {
			a.log.Info("[ARCHIVER] Archived %d recording frames to %s", count, parquetFilename)
			// Sovereign Grade: Reclaim disk space
			a.db.Exec(`VACUUM`)
		}
	} else {
		os.Remove(parquetFilePath) // Clean up empty file
	}
	return nil
}

func (a *Archiver) archiveTerminalLogs() error {
	// terminal_logs uses DATETIME strings, not unix timestamps
	cutoff := time.Now().Add(-a.retention).Format("2006-01-02 15:04:05")

	rows, err := a.db.Query(`SELECT timestamp, tenant_id, session_id, host, output FROM terminal_logs WHERE timestamp < ?`, cutoff)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	parquetFilename := fmt.Sprintf("logs_%s.parquet", time.Now().Format("2006-01-02_15-04-05"))
	parquetFilePath := filepath.Join(a.archivePath, parquetFilename)

	f, err := os.Create(parquetFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := parquet.NewGenericWriter[ParquetLogEntry](f, parquet.Compression(&parquet.Zstd))
	var batch []ParquetLogEntry

	for rows.Next() {
		var ts, tenant, sess, host, output string
		if err := rows.Scan(&ts, &tenant, &sess, &host, &output); err != nil {
			continue
		}
		batch = append(batch, ParquetLogEntry{ts, tenant, sess, host, output})
		count++
		if len(batch) >= 5000 {
			writer.Write(batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		writer.Write(batch)
	}
	writer.Close()

	if count > 0 {
		a.db.Exec(`DELETE FROM terminal_logs WHERE timestamp < ?`, cutoff)
		if a.log != nil {
			a.log.Info("[ARCHIVER] Archived %d terminal logs to %s", count, parquetFilename)
			// Sovereign Grade: Reclaim disk space
			a.db.Exec(`VACUUM`)
		}
	} else {
		os.Remove(parquetFilePath) // Clean up empty file
	}
	return nil
}
