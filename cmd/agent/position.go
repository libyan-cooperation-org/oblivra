package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Position tracks how far we've tailed in each file. Persisted as JSON in
// `<stateDir>/positions.json`. On agent restart we reopen each file at the
// recorded offset so we don't re-emit events the platform already has.
//
// Key is the absolute file path. Inode-based tracking (to detect
// log-rotated files where the path was reused) is future work; for now we
// detect rotation by comparing recorded size vs current size: if the file
// shrunk, we treat it as rotated and re-tail from 0.
type Position struct {
	Path  string `json:"path"`
	Inode uint64 `json:"inode,omitempty"`
	Size  int64  `json:"size"`
	Off   int64  `json:"offset"`
}

type PositionStore struct {
	path string

	mu  sync.Mutex
	all map[string]Position
}

func NewPositionStore(stateDir string) (*PositionStore, error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	p := &PositionStore{
		path: filepath.Join(stateDir, "positions.json"),
		all:  map[string]Position{},
	}
	if err := p.load(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PositionStore) load() error {
	body, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(body) == 0 {
		return nil
	}
	var arr []Position
	if err := json.Unmarshal(body, &arr); err != nil {
		return fmt.Errorf("positions.json corrupt: %w", err)
	}
	for _, pos := range arr {
		p.all[pos.Path] = pos
	}
	return nil
}

func (p *PositionStore) Get(path string) (Position, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pos, ok := p.all[path]
	return pos, ok
}

// Set updates the recorded position and persists immediately. Persistence
// is synchronous and atomic-rename so a crash between tail-loop iterations
// doesn't lose the offset.
func (p *PositionStore) Set(pos Position) error {
	p.mu.Lock()
	p.all[pos.Path] = pos
	snapshot := make([]Position, 0, len(p.all))
	for _, v := range p.all {
		snapshot = append(snapshot, v)
	}
	p.mu.Unlock()
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}

func (p *PositionStore) All() []Position {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Position, 0, len(p.all))
	for _, v := range p.all {
		out = append(out, v)
	}
	return out
}
