package app

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/sharing"
)

// localSession holds a single terminal process and its I/O handles.
type localSession struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	pty    *ptySession // platform PTY abstraction (nil until started)
	cancel context.CancelFunc
}

// LocalService manages local terminal sessions.
type LocalService struct {
	BaseService
	ctx        context.Context
	bus        *eventbus.Bus
	log        *logger.Logger
	sessionMgr SessionOperations
	recManager sharing.RecordingProvider

	mu       *sync.Mutex
	sessions map[string]*localSession
	batchers *sync.Map // Map of sessionID -> *OutputBatcher
}

func (s *LocalService) Name() string { return "LocalService" }

func NewLocalService(bus *eventbus.Bus, log *logger.Logger, sessionMgr SessionOperations, recManager sharing.RecordingProvider) *LocalService {
	return &LocalService{
		bus:        bus,
		log:        log.WithPrefix("local-term"),
		sessionMgr: sessionMgr,
		recManager: recManager,
		sessions:   make(map[string]*localSession),
		mu:         &sync.Mutex{},
		batchers:   &sync.Map{},
	}
}

func (s *LocalService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("LocalService started (platform: %s)", runtime.GOOS)
}

// StartLocalSession creates and starts a new local terminal session.
// On Linux/macOS this spawns a real PTY via creack/pty for full terminal
// emulation (cursor, colours, resize). On Windows it falls back to pipes.
func (s *LocalService) StartLocalSession() (string, error) {
	id := uuid.New().String()
	s.log.Info("Starting local session %s", id)

	// Pick the right shell
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NoExit")
	default:
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		cmd = exec.Command(shell, "-l")
	}

	// Use the platform-specific PTY abstraction (creack/pty on unix, pipes on windows)
	ps, err := startPTY(cmd, 120, 40)
	if err != nil {
		return "", fmt.Errorf("start pty session: %w", err)
	}

	// Per-session cancel context for read goroutines
	ctx, cancel := context.WithCancel(context.Background())

	sess := &localSession{cmd: cmd, stdin: ps.stdin, pty: ps, cancel: cancel}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	// Persist session
	s.sessionMgr.Create(database.Session{
		ID:     id,
		HostID: "local",
		Status: "active",
	})

	// IMPORTANT: Tell the frontend a session exists
	EmitEvent(s.ctx, "session:started", map[string]string{
		"id":     id,
		"hostId": "local",
		"label":  "Local Terminal",
	})

	// Pump stdout → frontend (single reader on unix PTY, merged on windows)
	go s.pump(ctx, id, ps.stdout)
	// Wait for exit → cleanup
	go s.waitExit(id, cmd)

	s.log.Info("Session %s ready (pid=%d, pty=%v)", id, cmd.Process.Pid, ps.resize != nil)
	return id, nil
}

// pump reads from a pipe/pty and emits base64-encoded output to the frontend.
func (s *LocalService) pump(ctx context.Context, sessionID string, r io.Reader) {
	scanner := bufio.NewReader(r)
	buf := make([]byte, 32*1024)

	// Fetch or create a batcher for this session
	var batcher *OutputBatcher
	if b, ok := s.batchers.Load(sessionID); ok {
		batcher = b.(*OutputBatcher)
	} else {
		batcher = NewOutputBatcher(s.ctx, sessionID)
		s.batchers.Store(sessionID, batcher)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := scanner.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])

			// Async recording
			if s.recManager != nil {
				go s.recManager.RecordOutput(sessionID, chunk)
			}

			// Batch output instead of emitting immediately to save IPC load
			batcher.Write(chunk)
		}
		if err != nil {
			return
		}
	}
}

// waitExit waits for the shell process to finish and cleans up.
func (s *LocalService) waitExit(sessionID string, cmd *exec.Cmd) {
	_ = cmd.Wait()
	time.Sleep(50 * time.Millisecond) // let buffered output flush
	s.cleanup(sessionID, true)
}

// SendInput writes user keystrokes to the shell's stdin (base64 encoded).
func (s *LocalService) SendInput(sessionID string, data string) error {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}

	if s.recManager != nil {
		go s.recManager.RecordInput(sessionID, decoded)
	}

	_, err = sess.stdin.Write(decoded)
	return err
}

// Resize changes the terminal dimensions. On unix with PTY this sends a real
// SIGWINCH signal. On Windows (pipe mode) this is a no-op.
func (s *LocalService) Resize(sessionID string, cols, rows int) error {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if sess.pty != nil && sess.pty.resize != nil {
		return sess.pty.resize(cols, rows)
	}

	return nil // pipe mode, no-op
}

// CloseSession kills the session.
func (s *LocalService) CloseSession(sessionID string) error {
	return s.cleanup(sessionID, false)
}

// cleanup tears down all resources for a session. Idempotent.
func (s *LocalService) cleanup(sessionID string, processExited bool) error {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	delete(s.sessions, sessionID)
	s.mu.Unlock()

	sess.cancel()

	// Close PTY-specific resources first
	if sess.pty != nil && sess.pty.closer != nil {
		sess.pty.closer()
	} else if sess.stdin != nil {
		sess.stdin.Close()
	}

	if !processExited && sess.cmd.Process != nil {
		sess.cmd.Process.Kill()
		sess.cmd.Wait()
	}

	// Clean up the output batcher to prevent memory leak
	if b, ok := s.batchers.LoadAndDelete(sessionID); ok {
		batcher := b.(*OutputBatcher)
		batcher.Flush()
	}

	s.sessionMgr.UpdateStatus(sessionID, "closed")
	EmitEvent(s.ctx, "session-closed-"+sessionID, sessionID)
	EmitEvent(s.ctx, "session:closed", sessionID)
	s.log.Info("Session %s cleaned up", sessionID)
	return nil
}

// ---- File operations for local FileBrowser ----

// validatePath prevents path traversal attacks by ensuring the resolved
// absolute path is within the user's home directory.
func (s *LocalService) validatePath(path string) (string, error) {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)
	home, _ := os.UserHomeDir()
	// Allow access within home directory and common system paths
	if !strings.HasPrefix(abs, filepath.Clean(home)) {
		s.log.Warn("Path traversal blocked: %s (outside %s)", abs, home)
		return "", fmt.Errorf("access denied: path %q is outside home directory", path)
	}
	return abs, nil
}

func (s *LocalService) ListDirectory(ctxID string, path string) ([]FileInfo, error) {
	safePath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(safePath)
	if err != nil {
		return nil, err
	}
	var result []FileInfo
	for _, e := range entries {
		info, _ := e.Info()
		if info == nil {
			continue
		}
		result = append(result, FileInfo{
			Name: e.Name(), Size: info.Size(), Mode: info.Mode().String(),
			ModTime: info.ModTime(), IsDir: e.IsDir(),
		})
	}
	return result, nil
}

func (s *LocalService) Mkdir(ctxID string, path string) error {
	safePath, err := s.validatePath(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(safePath, 0755)
}

func (s *LocalService) Rename(ctxID string, old, new string) error {
	safeOld, err := s.validatePath(old)
	if err != nil {
		return err
	}
	safeNew, err := s.validatePath(new)
	if err != nil {
		return err
	}
	return os.Rename(safeOld, safeNew)
}

func (s *LocalService) Remove(ctxID string, path string) error {
	safePath, err := s.validatePath(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(safePath)
}

func (s *LocalService) ReadFile(ctxID string, path string) (string, error) {
	safePath, err := s.validatePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return "", err
	}
	if info.Size() > 5*1024*1024 {
		return "", fmt.Errorf("file too large (%d bytes), maximum allowed for preview is 5MB. Use SFTP download instead", info.Size())
	}

	b, err := os.ReadFile(safePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (s *LocalService) WriteFile(ctxID string, path string, b64 string) error {
	safePath, err := s.validatePath(path)
	if err != nil {
		return err
	}
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	return os.WriteFile(safePath, b, 0644)
}
