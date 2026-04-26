package agent

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// TestRequestRestart_FiresShutdownThenExit verifies the standard
// happy-path: shutdown is called, then exitFn with RestartExitCode.
func TestRequestRestart_FiresShutdownThenExit(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})

	var shutdownCalled int32
	rm := NewRestartManager(log, func(ctx context.Context) error {
		atomic.AddInt32(&shutdownCalled, 1)
		return nil
	})

	var gotCode int32
	rm.SetExitFn(func(code int) {
		atomic.StoreInt32(&gotCode, int32(code))
	})

	rm.RequestRestart(RestartReasonUIRequest, 100*time.Millisecond)

	if atomic.LoadInt32(&shutdownCalled) != 1 {
		t.Errorf("shutdown should be called exactly once, got %d", shutdownCalled)
	}
	if got := atomic.LoadInt32(&gotCode); got != RestartExitCode {
		t.Errorf("exit code: got %d, want %d", got, RestartExitCode)
	}
}

// TestRequestRestart_Idempotent verifies that concurrent triggers
// fire shutdown only once.
func TestRequestRestart_Idempotent(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})

	var shutdownCalled int32
	rm := NewRestartManager(log, func(ctx context.Context) error {
		atomic.AddInt32(&shutdownCalled, 1)
		return nil
	})
	rm.SetExitFn(func(int) {})

	// Fire 5 concurrent restart requests; only one should win.
	done := make(chan struct{}, 5)
	for i := 0; i < 5; i++ {
		go func() {
			rm.RequestRestart(RestartReasonWatchdog, 100*time.Millisecond)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}
	if atomic.LoadInt32(&shutdownCalled) != 1 {
		t.Errorf("shutdown should be called exactly once across concurrent triggers, got %d", shutdownCalled)
	}
}

// TestRequestRestart_ProceedsOnShutdownError: even if the drain
// errors (e.g. WAL flush timeout), we still exit with the restart
// code. Otherwise a flaky drain would leave the agent stuck.
func TestRequestRestart_ProceedsOnShutdownError(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})

	rm := NewRestartManager(log, func(ctx context.Context) error {
		return errors.New("wal flush timeout")
	})

	var gotCode int32
	rm.SetExitFn(func(code int) {
		atomic.StoreInt32(&gotCode, int32(code))
	})

	rm.RequestRestart(RestartReasonHealthFailure, 100*time.Millisecond)

	if got := atomic.LoadInt32(&gotCode); got != RestartExitCode {
		t.Errorf("exit code on drain error: got %d, want %d", got, RestartExitCode)
	}
}

// TestRequestRestart_NilShutdownIsValid: tests/embedded use cases
// can pass nil shutdown for a "just exit" path.
func TestRequestRestart_NilShutdownIsValid(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})

	rm := NewRestartManager(log, nil)
	exited := make(chan int, 1)
	rm.SetExitFn(func(code int) { exited <- code })

	rm.RequestRestart(RestartReasonConfigChange, 50*time.Millisecond)

	select {
	case code := <-exited:
		if code != RestartExitCode {
			t.Errorf("exit code: got %d, want %d", code, RestartExitCode)
		}
	case <-time.After(time.Second):
		t.Fatal("expected exitFn to be called")
	}
}
