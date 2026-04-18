package isolation

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type WorkerType string

const (
	WorkerTypeDetection WorkerType = "detect"
	WorkerTypePolicy    WorkerType = "policy"
	WorkerTypeEnrich    WorkerType = "enrich"
)

type ProcessManager struct {
	mu      sync.RWMutex
	workers map[WorkerType]*exec.Cmd
	log     *logger.Logger
}

func NewProcessManager(l *logger.Logger) *ProcessManager {
	return &ProcessManager{
		workers: make(map[WorkerType]*exec.Cmd),
		log:     l,
	}
}

func (m *ProcessManager) StartWorker(ctx context.Context, wType WorkerType, executable string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workers[wType]; exists {
		return fmt.Errorf("worker %s is already running", wType)
	}

	cmd := exec.CommandContext(ctx, executable, "worker", "--type", string(wType))

	// Create pipes for stdout/stderr to capture logs from the isolated process
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout for worker %s: %w", wType, err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr for worker %s: %w", wType, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start worker %s: %w", wType, err)
	}

	m.workers[wType] = cmd

	// Monitor output in background
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			m.log.Info(fmt.Sprintf("[Worker-%s] %s", wType, scanner.Text()))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			m.log.Error(fmt.Sprintf("[Worker-%s] %s", wType, scanner.Text()))
		}
	}()

	// Monitor process exit
	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.workers, wType)
			m.mu.Unlock()
		}()

		err := cmd.Wait()
		if err != nil {
			m.log.Error(fmt.Sprintf("Worker %s exited with error: %v", wType, err))
		} else {
			m.log.Info(fmt.Sprintf("Worker %s exited successfully", wType))
		}
	}()

	return nil
}

func (m *ProcessManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for wType, cmd := range m.workers {
		if cmd.Process != nil {
			cmd.Process.Kill()
			m.log.Info(fmt.Sprintf("Killed worker: %s", wType))
		}
	}
}
