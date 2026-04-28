package services

// IOConfigService — wires the I/O pipeline framework into the
// container. Exposes the IOConfigProvider interface that the REST
// /api/v1/io/config endpoint reads.
//
// Why a service: the Pipeline's start/stop lifecycle needs to live
// alongside the other long-lived services so it gets the same
// graceful shutdown treatment. Implementing the kernel.Service
// interface (Name/Start/Stop/Dependencies) means we don't need to
// remember to manually wire it through the boot path.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	oblio "github.com/kingknull/oblivrashell/internal/io"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// IOConfigService is the api.IOConfigProvider implementation. It owns
// a single oblio.Pipeline and reads/writes the YAML config file the
// /connectors UI edits.
type IOConfigService struct {
	BaseService
	path string
	log  *logger.Logger

	mu       sync.Mutex
	pipeline *oblio.Pipeline
	cancel   context.CancelFunc
}

// NewIOConfigService constructs the service. `path` is the YAML file
// the operator edits via /connectors. Missing file is OK — the
// service returns an empty pipeline until the operator saves a config.
func NewIOConfigService(path string, log *logger.Logger) *IOConfigService {
	return &IOConfigService{
		path: path,
		log:  log.WithPrefix("io-config"),
	}
}

func (s *IOConfigService) Name() string         { return "io-config-service" }
func (s *IOConfigService) Dependencies() []string { return nil }

// Start is a no-op — pipeline boot is gated behind StartPipeline,
// which the container calls AFTER the REST server is wired so the
// /api/v1/io/stats endpoint reflects live counters from the moment
// the pipeline starts producing events.
func (s *IOConfigService) Start(ctx context.Context) error {
	return nil
}

func (s *IOConfigService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	if s.pipeline != nil {
		_ = s.pipeline.Stop(ctx)
		s.pipeline = nil
	}
	return nil
}

// StartPipeline reads the YAML config and starts the pipeline. Called
// by container.go after construction. Failure is logged but doesn't
// abort the rest of the platform — operators can fix the config via
// /connectors and trigger a hot-reload (or restart the binary).
func (s *IOConfigService) StartPipeline(parent context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := oblio.LoadConfig(s.path)
	if err != nil {
		return fmt.Errorf("load %s: %w", s.path, err)
	}
	if len(cfg.Inputs) == 0 && len(cfg.Outputs) == 0 {
		s.log.Info("no inputs/outputs configured at %s — pipeline idle. Edit via /connectors.", s.path)
		return nil
	}

	pipe := oblio.NewPipeline(s.log)
	for _, inCfg := range cfg.Inputs {
		in, err := oblio.NewInput(inCfg, s.log)
		if err != nil {
			return fmt.Errorf("input %q: %w", inCfg.ID, err)
		}
		pipe.AddInput(in)
	}
	for _, outCfg := range cfg.Outputs {
		out, err := oblio.NewOutput(outCfg, s.log)
		if err != nil {
			return fmt.Errorf("output %q: %w", outCfg.ID, err)
		}
		pipe.AddOutput(out)
	}

	pipeCtx, cancel := context.WithCancel(parent)
	if err := pipe.Start(pipeCtx); err != nil {
		cancel()
		return err
	}
	s.pipeline = pipe
	s.cancel = cancel
	s.log.Info("pipeline started: %d input(s), %d output(s)", len(cfg.Inputs), len(cfg.Outputs))
	return nil
}

// ── api.IOConfigProvider implementation ──────────────────────────

func (s *IOConfigService) ReadFile() ([]byte, error) {
	return os.ReadFile(s.path)
}

func (s *IOConfigService) WriteFile(yaml []byte) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return err
	}
	// Atomic-ish write: same dir + rename. Avoids partial-file race
	// with the fsnotify watcher.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, yaml, 0640); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *IOConfigService) PipelineStats() (uint64, uint64, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pipeline == nil {
		return 0, 0, 0
	}
	return s.pipeline.Stats()
}
