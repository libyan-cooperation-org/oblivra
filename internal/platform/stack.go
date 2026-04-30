// Package platform constructs the OBLIVRA service stack — WAL, hot store,
// ingest pipeline, and the user-facing services that wrap them. Both the
// Wails desktop binary and the headless server bootstrap through here, so
// the layout stays identical across surfaces.
package platform

import (
	"fmt"
	"log/slog"

	"github.com/kingknull/oblivra/internal/datapath"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/services"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/wal"
)

type Stack struct {
	Log    *slog.Logger
	System *services.SystemService
	Siem   *services.SiemService

	pipeline *ingest.Pipeline
	hot      *hot.Store
	wal      *wal.WAL
	search   *search.Index
}

type Options struct {
	Logger *slog.Logger
	// InMemory uses a memory-backed Badger and skips on-disk WAL persistence —
	// handy for tests.
	InMemory bool
}

func New(opts Options) (*Stack, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	dir, err := datapath.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	walDir, err := datapath.Sub(dir, "wal")
	if err != nil {
		return nil, fmt.Errorf("wal dir: %w", err)
	}
	w, err := wal.Open(wal.Options{Dir: walDir, SyncOnAppend: true})
	if err != nil {
		return nil, fmt.Errorf("open wal: %w", err)
	}

	hotDir, err := datapath.Sub(dir, "siem_hot.badger")
	if err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("hot dir: %w", err)
	}
	store, err := hot.Open(hot.Options{Dir: hotDir, InMemory: opts.InMemory})
	if err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("open hot store: %w", err)
	}

	indexDir, err := datapath.Sub(dir, "bleve.idx")
	if err != nil {
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("index dir: %w", err)
	}
	idx, err := search.Open(search.Options{Dir: indexDir, InMemory: opts.InMemory})
	if err != nil {
		_ = store.Close()
		_ = w.Close()
		return nil, fmt.Errorf("open search index: %w", err)
	}

	pipeline := ingest.New(opts.Logger, w, store, idx)

	system := services.NewSystemService(opts.Logger)
	siem := services.NewSiemService(opts.Logger, pipeline)

	opts.Logger.Info("platform ready", "dataDir", dir)

	return &Stack{
		Log:      opts.Logger,
		System:   system,
		Siem:     siem,
		pipeline: pipeline,
		hot:      store,
		wal:      w,
		search:   idx,
	}, nil
}

// Close releases all underlying resources.
func (s *Stack) Close() error {
	var first error
	if s.search != nil {
		if err := s.search.Close(); err != nil && first == nil {
			first = err
		}
	}
	if s.hot != nil {
		if err := s.hot.Close(); err != nil && first == nil {
			first = err
		}
	}
	if s.wal != nil {
		if err := s.wal.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
