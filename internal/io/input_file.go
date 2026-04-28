package io

// File input — tail a file or a glob of files.
//
// Behaviour:
//   • Glob is expanded once at start and re-checked every 30s so
//     newly-created files matching the pattern auto-attach.
//   • Each file is tailed line-by-line. Position survives in-place
//     log rotation (we re-open by path when the inode changes).
//   • `start_at: beginning|end` — default `end` ("ship new lines, not
//     the existing 4 GB backlog").
//
// Knowingly out of scope (defer to v2):
//   • Multi-line events. Operators put one event per line. We could
//     add a `multiline_pattern` regex later.
//   • Compression-handled tailing (gz/bz2). Use `exec` input to
//     gunzip into a fifo if needed.

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"gopkg.in/yaml.v3"
)

type fileInputConfig struct {
	Paths      []string `yaml:"paths"`
	Sourcetype string   `yaml:"sourcetype"`
	StartAt    string   `yaml:"start_at"` // "beginning" | "end"
	Host       string   `yaml:"host"`     // override; defaults to os.Hostname()
}

type FileInput struct {
	id   string
	cfg  fileInputConfig
	log  *logger.Logger
	host string

	mu      sync.Mutex
	tailing map[string]*tailedFile
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	scanInt time.Duration
}

type tailedFile struct {
	path  string
	inode uint64
	pos   int64
}

func NewFileInput(id string, raw map[string]interface{}, log *logger.Logger) (*FileInput, error) {
	cfg, err := decodeYAMLMap[fileInputConfig](raw)
	if err != nil {
		return nil, fmt.Errorf("input file %q: %w", id, err)
	}
	if len(cfg.Paths) == 0 {
		return nil, fmt.Errorf("input file %q: at least one path is required", id)
	}
	if cfg.StartAt == "" {
		cfg.StartAt = "end"
	}
	if cfg.StartAt != "beginning" && cfg.StartAt != "end" {
		return nil, fmt.Errorf("input file %q: start_at must be 'beginning' or 'end' (got %q)", id, cfg.StartAt)
	}
	host := cfg.Host
	if host == "" {
		host, _ = os.Hostname()
	}
	return &FileInput{
		id:      id,
		cfg:     cfg,
		log:     log.WithPrefix("input.file"),
		host:    host,
		tailing: map[string]*tailedFile{},
		scanInt: 30 * time.Second,
	}, nil
}

func (f *FileInput) Name() string { return f.id }
func (f *FileInput) Type() string { return "file" }

func (f *FileInput) Start(ctx context.Context, out chan<- Event) error {
	pluginCtx, cancel := context.WithCancel(ctx)
	f.cancel = cancel

	f.wg.Add(1)
	go f.scanLoop(pluginCtx, out)
	return nil
}

func (f *FileInput) Stop() error {
	if f.cancel != nil {
		f.cancel()
	}
	done := make(chan struct{})
	go func() { f.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		f.log.Warn("file input %q stop timed out", f.id)
	}
	return nil
}

// scanLoop walks each glob entry, attaches new tailers for previously-
// unseen files, and detaches tailers whose underlying file is gone.
func (f *FileInput) scanLoop(ctx context.Context, out chan<- Event) {
	defer f.wg.Done()

	f.rescanGlobs(ctx, out)

	t := time.NewTicker(f.scanInt)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			f.rescanGlobs(ctx, out)
		}
	}
}

func (f *FileInput) rescanGlobs(ctx context.Context, out chan<- Event) {
	seen := map[string]struct{}{}
	for _, pattern := range f.cfg.Paths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			f.log.Warn("glob %q: %v", pattern, err)
			continue
		}
		for _, m := range matches {
			seen[m] = struct{}{}
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for path := range seen {
		if _, ok := f.tailing[path]; ok {
			continue
		}
		t := &tailedFile{path: path}
		f.tailing[path] = t
		f.wg.Add(1)
		go f.tailFile(ctx, t, out)
		f.log.Info("attached tailer to %s", path)
	}
	for path := range f.tailing {
		if _, ok := seen[path]; !ok {
			delete(f.tailing, path)
			f.log.Info("detached tailer from %s (file no longer matches glob)", path)
		}
	}
}

// tailFile is the per-file goroutine. Opens the file, seeks per
// `start_at`, then loops reading new lines. On rotation (inode change)
// it re-opens at offset 0.
func (f *FileInput) tailFile(ctx context.Context, t *tailedFile, out chan<- Event) {
	defer f.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fh, err := os.Open(t.path)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}

		st, err := fh.Stat()
		if err != nil {
			fh.Close()
			return
		}
		newInode := statInode(st)
		if t.inode == 0 {
			t.inode = newInode
			if f.cfg.StartAt == "end" {
				t.pos, _ = fh.Seek(0, 2)
			}
		} else if newInode != t.inode {
			// Rotated — reset to start of new file.
			t.inode = newInode
			t.pos = 0
		}

		_, _ = fh.Seek(t.pos, 0)
		reader := bufio.NewReader(fh)

		for {
			select {
			case <-ctx.Done():
				fh.Close()
				return
			default:
			}
			line, err := reader.ReadString('\n')
			if err != nil {
				t.pos, _ = fh.Seek(0, 1)
				break
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}
			ev := Event{
				Timestamp:  time.Now().UTC(),
				Source:     "file:" + t.path,
				Sourcetype: f.cfg.Sourcetype,
				Host:       f.host,
				Raw:        line,
				InputID:    f.id,
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				fh.Close()
				return
			}
		}

		fh.Close()
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// decodeYAMLMap re-marshals a generic map and unmarshals into the
// target type. Used because PluginConfig.Raw arrives as
// map[string]interface{} but we want type-safe access.
func decodeYAMLMap[T any](raw map[string]interface{}) (T, error) {
	var zero T
	b, err := yaml.Marshal(raw)
	if err != nil {
		return zero, err
	}
	var out T
	if err := yaml.Unmarshal(b, &out); err != nil {
		return zero, err
	}
	return out, nil
}
