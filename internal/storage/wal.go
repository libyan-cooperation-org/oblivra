package storage

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// WAL (Write-Ahead Log) provides crash-safe durability for incoming log streams
// before they are fully indexed into Bleve/SQLite/BadgerDB.
type WAL struct {
	mu           sync.Mutex
	file         *os.File
	filename     string
	writer       *bufio.Writer
	log          *logger.Logger
	flushedBytes int64
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewWAL creates or opens an existing Write-Ahead Log in the given data directory.
func NewWAL(dataDir string, log *logger.Logger) (*WAL, error) {
	walDir := filepath.Join(dataDir, "wal")
	// CS-25: Restrict permissions to owner only (0700) to prevent local exposure.
	if err := os.MkdirAll(walDir, 0700); err != nil {
		return nil, fmt.Errorf("create wal dir: %w", err)
	}

	walFile := filepath.Join(walDir, "ingest.wal")

	f, err := os.OpenFile(walFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open wal file: %w", err)
	}

	info, err := f.Stat()
	var size int64
	if err == nil {
		size = info.Size()
	}

	if log != nil {
		log.Info("[STORAGE] Opened WAL: %s (Size: %d bytes)", walFile, size)
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := &WAL{
		file:         f,
		filename:     walFile,
		writer:       bufio.NewWriterSize(f, 32*1024), // 32KB buffer for speed
		log:          log,
		flushedBytes: size,
		ctx:          ctx,
		cancel:       cancel,
	}

	go w.syncLoop()
	return w, nil
}

// Append writes a raw byte payload to the end of the WAL.
// Structure: [Length: 4B][Checksum: 4B][Payload: NB]
func (w *WAL) Append(payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 1. Calculate Checksum
	checksum := crc32.ChecksumIEEE(payload)

	// 2. Write Length Header (4 bytes)
	pLen := len(payload)
	// G115: Prevent overflow. Reject payloads > 10MB.
	if pLen > 10*1024*1024 {
		return fmt.Errorf("payload too large: %d bytes (max 10MB)", pLen)
	}
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(pLen))

	if _, err := w.writer.Write(lenBuf); err != nil {
		return err
	}

	// 3. Write Checksum Header (4 bytes)
	checkBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(checkBuf, checksum)
	if _, err := w.writer.Write(checkBuf); err != nil {
		return err
	}

	// 4. Write Data
	if _, err := w.writer.Write(payload); err != nil {
		return err
	}

	w.flushedBytes += int64(8 + len(payload))
	// Flush to the OS buffer immediately to ensure crash resilience 
	// (syncLoop will handle the physical disk sync every 50ms)
	return w.writer.Flush()
}

func (w *WAL) syncLoop() {
	ticker := time.NewTicker(50 * time.Millisecond) // Sovereign grade: 50ms durability window
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.Sync()
		}
	}
}

// Sync forces the buffered writes to disk, ensuring crash durability.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Sync()
}

// Checkpoint deletes the current WAL file.
// This is called when the ingestion pipeline successfully parses and dual-writes all pending WAL entries.
func (w *WAL) Checkpoint() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Flush and close current file
	w.writer.Flush()
	w.file.Close()

	// Truncate the file payload by recreating it empty
	f, err := os.OpenFile(w.filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("checkpoint reopen failed: %w", err)
	}

	w.file = f
	w.writer.Reset(f)
	w.flushedBytes = 0

	if w.log != nil {
		w.log.Debug("[STORAGE] WAL checkpointed (truncated)")
	}

	return nil
}

// Replay reads the entire WAL from the beginning and calls the provided function for each payload.
// If the function returns an error, the replay halts.
func (w *WAL) Replay(fn func(payload []byte) error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Flush anything pending to disk before reading
	w.writer.Flush()

	// We open a separate read handler to not interrupt the write pointer position
	rf, err := os.Open(w.filename)
	if err != nil {
		return fmt.Errorf("open wal for replay: %w", err)
	}
	defer rf.Close()

	reader := bufio.NewReader(rf)
	count := 0

	for {
		// 1. Read Length Header
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			if err == io.EOF {
				break // Normal end of file
			}
			return fmt.Errorf("read header error (corrupt wal?): %w", err)
		}

		length := binary.LittleEndian.Uint32(lenBuf)

		// Defense-in-depth: Prevent physical byte corruption from allocating MaxUint32
		// and instantly crashing the daemon via OOM. Reject payloads > 10MB.
		if length > 10*1024*1024 {
			if w.log != nil {
				w.log.Warn("[STORAGE] WAL corruption detected: payload length %d exceeds 10MB limit. Skipping.", length)
			}
			continue // Skip corrupted entry rather than crashing
		}

		// 2. Read Checksum Header
		checkBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, checkBuf); err != nil {
			return fmt.Errorf("read checksum error (corrupt wal?): %w", err)
		}
		expectedChecksum := binary.LittleEndian.Uint32(checkBuf)

		// 3. Read Payload
		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return fmt.Errorf("read payload error (corrupt wal?): %w", err)
		}

		// 4. Verify Checksum
		// 4. Verify Checksum
		actualChecksum := crc32.ChecksumIEEE(payload)
		if actualChecksum != expectedChecksum {
			if w.log != nil {
				w.log.Error("[STORAGE] Checksum mismatch on record %d: expected %x, got %x (skipping corrupt entry)", count, expectedChecksum, actualChecksum)
			}
			continue
		}

		// 5. Invoke user function
		if err := fn(payload); err != nil {
			return fmt.Errorf("replay func failed on record %d: %w", count, err)
		}

		count++
	}

	if count > 0 && w.log != nil {
		w.log.Info("[STORAGE] Replayed %d records from WAL upon startup", count)
	}

	return nil
}

// ReadWALManual provides low-level read access to a WAL file without requiring a full WAL manager instance.
// Useful for replay tools and forensic analysis.
func ReadWALManual(filename string, fn func(payload []byte) error) error {
	rf, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open wal manual: %w", err)
	}
	defer rf.Close()

	reader := bufio.NewReader(rf)
	count := 0

	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, lenBuf); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read header error: %w", err)
		}

		length := binary.LittleEndian.Uint32(lenBuf)
		if length > 10*1024*1024 {
			return fmt.Errorf("corrupt wal: length %d too large", length)
		}

		checkBuf := make([]byte, 4)
		if _, err := io.ReadFull(reader, checkBuf); err != nil {
			return fmt.Errorf("read checksum error: %w", err)
		}
		expectedChecksum := binary.LittleEndian.Uint32(checkBuf)

		payload := make([]byte, length)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return fmt.Errorf("read payload error: %w", err)
		}

		actualChecksum := crc32.ChecksumIEEE(payload)
		if actualChecksum != expectedChecksum {
			return fmt.Errorf("checksum mismatch on record %d", count)
		}

		if err := fn(payload); err != nil {
			return err
		}

		count++
	}

	return nil
}

// Close gracefully flushes and shuts down the WAL.
func (w *WAL) Close() error {
	w.cancel()
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.writer != nil {
		w.writer.Flush()
	}
	if w.file != nil {
		w.file.Sync()
		return w.file.Close()
	}
	return nil
}
