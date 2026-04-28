package io

// File output — write events to a local file as newline-delimited
// JSON. Rotates by size or time.
//
// Use cases:
//   • Air-gap "sneakernet" — write to a USB-mounted directory,
//     someone walks the file across the airgap to ingest later
//   • Debug — operators tail this when investigating a flaky input
//   • Side-channel archive — compliance retention without paying for S3
//
// Config:
//
//   - id: archive
//     type: file
//     path: "/var/log/oblivra/events-%Y%m%d-%H.json"
//     rotate: "1h"            # | "100MB" | "1d"
//     compression: gzip       # optional; rotated files get .gz
//     mode: 0640              # default

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type fileOutputConfig struct {
	Path        string `yaml:"path"`
	Rotate      string `yaml:"rotate"`      // "1h" / "1d" / "100MB"
	Compression string `yaml:"compression"` // "" or "gzip"
	Mode        uint32 `yaml:"mode"`
}

type FileOutput struct {
	id  string
	cfg fileOutputConfig
	log *logger.Logger

	rotateBy   string // "time" | "size"
	rotateDur  time.Duration
	rotateSize int64

	mu          sync.Mutex
	fh          *os.File
	currentPath string
	openedAt    time.Time
	bytesWritten int64
}

func NewFileOutputReal(id string, raw map[string]interface{}, log *logger.Logger) (*FileOutput, error) {
	cfg, err := decodeYAMLMap[fileOutputConfig](raw)
	if err != nil {
		return nil, fmt.Errorf("output file %q: %w", id, err)
	}
	if cfg.Path == "" {
		return nil, fmt.Errorf("output file %q: path is required", id)
	}
	if cfg.Mode == 0 {
		cfg.Mode = 0640
	}
	f := &FileOutput{
		id:  id,
		cfg: cfg,
		log: log.WithPrefix("output.file"),
	}
	if err := f.parseRotate(); err != nil {
		return nil, fmt.Errorf("output file %q: %w", id, err)
	}
	return f, nil
}

func (f *FileOutput) parseRotate() error {
	r := strings.TrimSpace(f.cfg.Rotate)
	if r == "" {
		// Default: rotate hourly.
		f.rotateBy = "time"
		f.rotateDur = time.Hour
		return nil
	}
	if strings.HasSuffix(r, "MB") || strings.HasSuffix(r, "mb") || strings.HasSuffix(r, "MiB") {
		num := strings.TrimRight(r, "MmiBb")
		n, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return fmt.Errorf("rotate size %q: %w", r, err)
		}
		f.rotateBy = "size"
		f.rotateSize = n * 1024 * 1024
		return nil
	}
	d, err := time.ParseDuration(r)
	if err != nil {
		// Last try — single-letter suffix like "1h", "1d".
		switch {
		case strings.HasSuffix(r, "d"):
			n, e := strconv.Atoi(strings.TrimSuffix(r, "d"))
			if e != nil {
				return fmt.Errorf("rotate duration %q: %w", r, err)
			}
			d = time.Duration(n) * 24 * time.Hour
		default:
			return fmt.Errorf("rotate duration %q: %w", r, err)
		}
	}
	f.rotateBy = "time"
	f.rotateDur = d
	return nil
}

func (f *FileOutput) Name() string { return f.id }
func (f *FileOutput) Type() string { return "file" }

func (f *FileOutput) Write(_ context.Context, ev Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.ensureOpen(); err != nil {
		return err
	}
	if f.shouldRotate() {
		if err := f.rotateNow(); err != nil {
			return err
		}
	}
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	n, err := f.fh.Write(body)
	if err != nil {
		return err
	}
	f.bytesWritten += int64(n)
	return nil
}

func (f *FileOutput) Flush(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fh == nil {
		return nil
	}
	return f.fh.Sync()
}

func (f *FileOutput) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fh != nil {
		_ = f.fh.Sync()
		_ = f.fh.Close()
		f.fh = nil
	}
	return nil
}

// ensureOpen opens the current target file (if not already open)
// after expanding strftime-style %Y/%m/%d/%H tokens in the path.
func (f *FileOutput) ensureOpen() error {
	if f.fh != nil {
		return nil
	}
	path := expandPath(f.cfg.Path, time.Now())
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.FileMode(f.cfg.Mode))
	if err != nil {
		return err
	}
	f.fh = fh
	f.currentPath = path
	f.openedAt = time.Now()
	st, _ := fh.Stat()
	if st != nil {
		f.bytesWritten = st.Size()
	}
	return nil
}

func (f *FileOutput) shouldRotate() bool {
	switch f.rotateBy {
	case "time":
		return time.Since(f.openedAt) >= f.rotateDur
	case "size":
		return f.bytesWritten >= f.rotateSize
	}
	return false
}

func (f *FileOutput) rotateNow() error {
	if f.fh != nil {
		_ = f.fh.Sync()
		_ = f.fh.Close()
		f.fh = nil
	}
	// gzip the just-closed file if compression is enabled.
	if f.cfg.Compression == "gzip" && f.currentPath != "" {
		go gzipFile(f.currentPath, f.log)
	}
	f.bytesWritten = 0
	return f.ensureOpen()
}

// expandPath replaces %Y/%m/%d/%H/%M/%S with the corresponding
// time fields. Used so daily/hourly rotation can be encoded directly
// in the path template.
func expandPath(template string, t time.Time) string {
	r := strings.NewReplacer(
		"%Y", t.UTC().Format("2006"),
		"%m", t.UTC().Format("01"),
		"%d", t.UTC().Format("02"),
		"%H", t.UTC().Format("15"),
		"%M", t.UTC().Format("04"),
		"%S", t.UTC().Format("05"),
	)
	return r.Replace(template)
}

func gzipFile(src string, log *logger.Logger) {
	in, err := os.Open(src)
	if err != nil {
		log.Warn("gzip open %s: %v", src, err)
		return
	}
	defer in.Close()
	out, err := os.Create(src + ".gz.tmp")
	if err != nil {
		log.Warn("gzip create %s: %v", src, err)
		return
	}
	gz := gzip.NewWriter(out)
	if _, err := copyAll(gz, in); err != nil {
		log.Warn("gzip copy %s: %v", src, err)
		out.Close()
		return
	}
	if err := gz.Close(); err != nil {
		log.Warn("gzip close %s: %v", src, err)
		out.Close()
		return
	}
	out.Close()
	_ = os.Rename(src+".gz.tmp", src+".gz")
	_ = os.Remove(src)
}

// copyAll is a tiny re-implementation of io.Copy to avoid name
// collision with our own io package.
func copyAll(dst interface{ Write(p []byte) (int, error) }, src interface{ Read(p []byte) (int, error) }) (int64, error) {
	buf := make([]byte, 32*1024)
	var n int64
	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			n += int64(nw)
			if werr != nil {
				return n, werr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				return n, nil
			}
			return n, err
		}
	}
}
