package sigma

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a Sigma rule directory and calls onChange when any *.yml /
// *.yaml file is created, modified, or removed. Events are debounced to one
// fire per 500ms so a multi-file edit doesn't trigger ten reloads.
type Watcher struct {
	log     *slog.Logger
	dir     string
	debounce time.Duration

	mu      sync.Mutex
	cancel  func()
	stopped chan struct{}
}

func NewWatcher(log *slog.Logger, dir string) *Watcher {
	return &Watcher{log: log, dir: dir, debounce: 500 * time.Millisecond}
}

// Start begins watching. onChange is called from a goroutine — keep it short
// or delegate. Multiple calls before Stop are no-ops.
func (w *Watcher) Start(parent context.Context, onChange func()) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	if w.dir == "" {
		return nil // nothing to watch — silent no-op
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := fsw.Add(w.dir); err != nil {
		_ = fsw.Close()
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.stopped = make(chan struct{})

	go func() {
		defer close(w.stopped)
		defer fsw.Close()

		var pending bool
		var timer *time.Timer
		fire := func() {
			if onChange != nil {
				onChange()
			}
			pending = false
		}
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-fsw.Events:
				if !ok {
					return
				}
				if !relevant(ev.Name) {
					continue
				}
				if !pending {
					pending = true
					if timer == nil {
						timer = time.AfterFunc(w.debounce, fire)
					} else {
						timer.Reset(w.debounce)
					}
				} else {
					timer.Reset(w.debounce)
				}
			case err, ok := <-fsw.Errors:
				if !ok {
					return
				}
				w.log.Warn("sigma watcher", "err", err)
			}
		}
	}()
	w.log.Info("sigma watcher started", "dir", w.dir)
	return nil
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	stopped := w.stopped
	w.cancel = nil
	w.stopped = nil
	w.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-stopped
}

func relevant(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}
