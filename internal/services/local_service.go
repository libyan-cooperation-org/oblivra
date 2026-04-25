package services

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
	
	commandHistory *CommandHistoryService
	commandBuffers  map[string][]byte
	cmdMu           *sync.Mutex
}

func (s *LocalService) Name() string { return "local-service" }

// Dependencies returns service dependencies
func (s *LocalService) Dependencies() []string {
	return []string{}
}

func NewLocalService(bus *eventbus.Bus, log *logger.Logger, sessionMgr SessionOperations, recManager sharing.RecordingProvider) *LocalService {
	return &LocalService{
		bus:        bus,
		log:        log.WithPrefix("local-term"),
		sessionMgr: sessionMgr,
		recManager: recManager,
		sessions:   make(map[string]*localSession),
		mu:         &sync.Mutex{},
		batchers:   &sync.Map{},
		commandBuffers:   make(map[string][]byte),
		cmdMu:            &sync.Mutex{},
	}
}

func (s *LocalService) SetCommandHistory(svc *CommandHistoryService) {
	s.commandHistory = svc
}

func (s *LocalService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("LocalService started (platform: %s)", runtime.GOOS)
	return nil
}

func (s *LocalService) Stop(ctx context.Context) error {
	return nil
}

// StartLocalSession creates and starts a new local terminal session.
// On Linux/macOS this spawns a real PTY via creack/pty for full terminal
// emulation (cursor, colours, resize). On Windows it falls back to pipes.
func (s *LocalService) StartLocalSession() (sessionID string, retErr error) {
	// Catch any panic from ConPTY / syscall layer so the whole app doesn't crash.
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("[LOCAL] StartLocalSession panic recovered: %v", r)
			retErr = fmt.Errorf("terminal startup panic: %v", r)
		}
	}()

	id := "local-" + uuid.New().String()
	s.log.Info("Starting local session %s", id)

	// Pick the right shell
	cmd := s.getShellCommand()

	// Use the platform-specific PTY abstraction (creack/pty on unix, pipes on windows)
	ps, err := startPTY(cmd, 120, 40)
	if err != nil {
		// ConPTY failed — try plain pipe mode as last resort
		s.log.Warn("[LOCAL] startPTY failed (%v), falling back to pipe mode", err)
		var pipeErr error
		ps, pipeErr = startPTYPipeMode(cmd)
		if pipeErr != nil {
			return "", fmt.Errorf("start pty session: %w (pipe fallback: %v)", err, pipeErr)
		}
	}

	// Per-session cancel context for read goroutines
	ctx, cancel := context.WithCancel(s.ctx)

	// Use the cmd from the ptySession — on Windows ConPTY mode this is a
	// bare wrapper cmd, not the original exec.Command we built above.
	actualCmd := ps.cmd
	if actualCmd == nil {
		actualCmd = cmd
	}

	sess := &localSession{cmd: actualCmd, stdin: ps.stdin, pty: ps, cancel: cancel}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	// Persist session
	s.sessionMgr.Create(s.ctx, database.Session{
		ID:     id,
		HostID: "local",
		Status: "active",
	})

	// IMPORTANT: Tell the frontend a session exists
	EmitEvent("session:started", map[string]string{
		"id":     id,
		"hostId": "local",
		"label":  "Local Terminal",
	})

	// Pump stdout → frontend (single reader on unix PTY, merged on windows)
	// Give the frontend a moment to mount the component and subscribe to the output channel
	// before we start pumping the initial shell prompt.
	go func() {
		time.Sleep(300 * time.Millisecond)
		s.pump(ctx, id, ps.stdout)
	}()
	// Wait for exit → cleanup
	go s.waitExit(id, actualCmd)

	pid := 0
	if actualCmd.Process != nil {
		pid = actualCmd.Process.Pid
	}
	s.log.Info("Session %s ready (pid=%d, pty=%v)", id, pid, ps.resize != nil)
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
	defer func() {
		if r := recover(); r != nil {
			s.log.Warn("[LOCAL] waitExit recovered from panic (session %s): %v", sessionID, r)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		// Process.Wait() is safe even when cmd.Start() was never called.
		// exec.Cmd.Wait() panics if Start() was not used — use the lower-level call.
		cmd.Process.Wait() //nolint:errcheck
	}
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

	// Command history tracking
	s.cmdMu.Lock()
	buf := s.commandBuffers[sessionID]
	for _, b := range decoded {
		if b == '\r' || b == '\n' {
			if len(buf) > 0 {
				cmd := string(buf)
				if s.commandHistory != nil {
					s.commandHistory.RecordCommand("local", cmd)
				}
			}
			buf = nil
		} else if b == 127 || b == 8 {
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		} else if b >= 32 {
			buf = append(buf, b)
		}
	}
	s.commandBuffers[sessionID] = buf
	s.cmdMu.Unlock()

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

	if sess.pty != nil && sess.pty.closer != nil {
		// PTY mode: closer owns all handles including the process handle.
		// Kill the process first (if still running), then let closer release handles.
		if !processExited && sess.cmd != nil && sess.cmd.Process != nil {
			sess.cmd.Process.Kill() //nolint:errcheck
		}
		sess.pty.closer() //nolint:errcheck
	} else {
		// Pipe mode: kill process, close stdin.
		if !processExited && sess.cmd != nil && sess.cmd.Process != nil {
			sess.cmd.Process.Kill()  //nolint:errcheck
			sess.cmd.Wait()          //nolint:errcheck
		}
		if sess.stdin != nil {
			sess.stdin.Close()
		}
	}

	// Clean up the output batcher to prevent memory leak
	if b, ok := s.batchers.LoadAndDelete(sessionID); ok {
		batcher := b.(*OutputBatcher)
		batcher.Flush()
	}

	s.sessionMgr.UpdateStatus(s.ctx, sessionID, "closed")
	EmitEvent("session-closed-"+sessionID, sessionID)
	EmitEvent("session:closed", sessionID)
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
			ModTime: info.ModTime().Format(time.RFC3339), IsDir: e.IsDir(),
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
