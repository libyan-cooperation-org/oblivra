package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/sharing"
	"github.com/kingknull/oblivrashell/internal/ssh"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// SSHService manages SSH connections and sessions
type SSHService struct {
	BaseService
	db               database.DatabaseStore
	vault            vault.Provider
	hosts            database.HostStore
	sessions         database.SessionStore
	creds            database.CredentialStore
	siem             database.SIEMStore
	ctx              context.Context
	bus              *eventbus.Bus
	log              *logger.Logger
	shareManager     *sharing.ShareManager
	recordingManager sharing.RecordingProvider
	telemetryManager *monitoring.TelemetryManager
	transferManager  TransferProvider
	sanitizer        *security.ShellSanitizer
	manager          *ssh.SessionManager
	tailingSvc       *TailingService
	commandHistory   *CommandHistoryService
	sessionPersist   *SessionPersistence

	onOutputCb func(sessionID, hostID, data string)
	batchers   *sync.Map 

	commandBuffers map[string][]byte
	cmdMu          *sync.Mutex
}

// GetSession implements SessionProvider
func (s *SSHService) GetSession(id string) (*ssh.Session, bool) {
	return s.manager.Get(id)
}

// NewSSHService creates a new SSH service
func NewSSHService(
	db database.DatabaseStore,
	vlt vault.Provider,
	h database.HostStore,
	s database.SessionStore,
	c database.CredentialStore,
	siem database.SIEMStore,
	bus *eventbus.Bus,
	log *logger.Logger,
	shareManager *sharing.ShareManager,
	recordingManager sharing.RecordingProvider,
	telemetryManager *monitoring.TelemetryManager,
	transferManager TransferProvider,
	sanitizer *security.ShellSanitizer,
	tailingSvc *TailingService,
) *SSHService {
	return &SSHService{
		db:               db,
		vault:            vlt,
		hosts:            h,
		sessions:         s,
		creds:            c,
		siem:             siem,
		bus:              bus,
		log:              log,
		shareManager:     shareManager,
		recordingManager: recordingManager,
		telemetryManager: telemetryManager,
		transferManager:  transferManager,
		sanitizer:        sanitizer,
		manager:          ssh.NewSessionManager(100),
		tailingSvc:       tailingSvc,
		batchers:         &sync.Map{},
		commandBuffers:   make(map[string][]byte),
		cmdMu:            &sync.Mutex{},
	}
}

func (s *SSHService) SetSnippetService(svc *SnippetService) {
	// Logic: This could be used for autonomous follow-ups
}

func (s *SSHService) SetCommandHistory(svc *CommandHistoryService) {
	s.commandHistory = svc
}

func (s *SSHService) SetSessionPersistence(svc *SessionPersistence) {
	s.sessionPersist = svc
}

func (s *SSHService) SetTransferManager(tm TransferProvider) {
	s.transferManager = tm
}

func (s *SSHService) Name() string { return "ssh-service" }

func (s *SSHService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *SSHService) Stop(ctx context.Context) error {
	s.CloseAll()
	return nil
}

func (s *SSHService) SendInput(sessionID, data string) error {
	sess, ok := s.manager.Get(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return sess.Write([]byte(data))
}

// fetchAndDecryptHostPassword fetches the encrypted password blob from the DB
// and decrypts it using the vault. Returns the plaintext bytes for use only
// during SSH connection setup. The caller is responsible for zeroing the slice.
func (s *SSHService) fetchAndDecryptHostPassword(ctx context.Context, hostID string) ([]byte, error) {
	encRaw, err := s.hosts.GetEncryptedPassword(ctx, hostID)
	if err != nil || encRaw == "" {
		return nil, err
	}
	blob, err := base64.StdEncoding.DecodeString(encRaw)
	if err != nil {
		return nil, fmt.Errorf("credential requires migration to vault encryption")
	}
	decrypted, err := s.vault.Decrypt(blob)
	if err != nil {
		return nil, fmt.Errorf("decrypt host password: %w", err)
	}
	return decrypted, nil
}

// ... rest of the methods ...

// Connect establishes an SSH connection to a host using a centralized session lifecycle.
func (s *SSHService) Connect(ctx context.Context, hostID string) (string, error) {
	if s.vault != nil && !s.vault.IsUnlocked() {
		return "", fmt.Errorf("vault is locked")
	}
	s.log.Info("Connecting to hostID: %s", hostID)

	host, err := s.hosts.GetByID(ctx, hostID)
	if err != nil {
		s.log.Error("Failed to fetch host %s: %v", hostID, err)
		return "", fmt.Errorf("fetch host: %w", err)
	}

	sshConfig := s.prepareSSHConfig(ctx, host)
	session := ssh.NewSession(hostID, host.Label, *sshConfig)

	s.registerSessionCallbacks(ctx, session, host)

	if err := s.manager.Add(session); err != nil {
		return "", fmt.Errorf("add session: %w", err)
	}

	// Persist to database
	dbSess := database.Session{
		ID:        session.ID,
		HostID:    hostID,
		StartedAt: session.StartedAt,
		Status:    "active",
	}
	_ = s.sessions.Create(ctx, &dbSess)

	if err := session.Start(); err != nil {
		s.log.Error("Failed to start session %s: %v", session.ID, err)
		s.manager.Remove(session.ID)
		return "", fmt.Errorf("start session: %w", err)
	}

	s.bus.Publish(eventbus.EventConnectionOpened, map[string]string{
		"id":     session.ID,
		"hostId": hostID,
		"label":  host.Label,
	})

	EmitEvent("session:started", map[string]string{
		"id":     session.ID,
		"hostId": hostID,
		"label":  host.Label,
	})

	go s.startTelemetryPolling(session.Ctx, session.ID)
	go s.startStackDiscovery(session.Ctx, session.ID, hostID)
	go s.startSecurityAudit(session.Ctx, session.ID, hostID)

	return session.ID, nil
}

// ConnectToSession is a tactical variant of Connect that uses a pre-defined session ID.
func (s *SSHService) ConnectToSession(ctx context.Context, hostID string, sessionID string) (string, error) {
	if s.vault != nil && !s.vault.IsUnlocked() {
		return "", fmt.Errorf("vault is locked")
	}

	// 1. If session already exists with this ID, close it first to allow pivoting to new target
	if existing, ok := s.manager.Get(sessionID); ok {
		s.log.Info("Pivoting: Closing existing session %s to target new host %s", sessionID, hostID)
		existing.Close()
		s.manager.Remove(sessionID)
	}

	s.log.Info("Pivoting to hostID: %s using sessionID: %s", hostID, sessionID)

	host, err := s.hosts.GetByID(ctx, hostID)
	if err != nil {
		s.log.Error("Failed to fetch host %s: %v", hostID, err)
		return "", fmt.Errorf("fetch host: %w", err)
	}

	sshConfig := s.prepareSSHConfig(ctx, host)
	session := ssh.NewSessionWithID(sessionID, hostID, host.Label, *sshConfig)

	s.registerSessionCallbacks(ctx, session, host)

	if err := s.manager.Add(session); err != nil {
		return "", fmt.Errorf("add session: %w", err)
	}

	// Persist to database
	dbSess := database.Session{
		ID:        session.ID,
		HostID:    hostID,
		StartedAt: session.StartedAt,
		Status:    "active",
	}
	_ = s.sessions.Create(ctx, &dbSess)

	if err := session.Start(); err != nil {
		s.log.Error("Failed to start session %s: %v", session.ID, err)
		s.manager.Remove(session.ID)
		return "", fmt.Errorf("start session: %w", err)
	}

	s.bus.Publish(eventbus.EventConnectionOpened, map[string]string{
		"id":     session.ID,
		"hostId": hostID,
		"label":  host.Label,
	})

	EmitEvent("session:started", map[string]string{
		"id":     session.ID,
		"hostId": hostID,
		"label":  host.Label,
	})

	go s.startTelemetryPolling(session.Ctx, session.ID)
	go s.startStackDiscovery(session.Ctx, session.ID, hostID)
	go s.startSecurityAudit(session.Ctx, session.ID, hostID)

	return session.ID, nil
}

func (s *SSHService) prepareSSHConfig(ctx context.Context, host *database.Host) *ssh.ConnectionConfig {
	cfg := ssh.DefaultConfig()
	cfg.Host = host.Hostname
	if host.Port != 0 {
		cfg.Port = host.Port
	}
	cfg.Username = host.Username

	// SECURITY: Password is NOT on the host DTO (it's stripped before frontend serialisation).
	if host.HasPassword && s.vault != nil && s.vault.IsUnlocked() {
		if encRaw, err := s.fetchAndDecryptHostPassword(ctx, host.ID); err == nil {
			cfg.Password = encRaw
		}
	}

	// Logic: If a managed credential is linked, override defaults
	if host.CredentialID != "" && s.vault.IsUnlocked() {
		cred, err := s.creds.GetByID(ctx, host.CredentialID)
		if err == nil {
			decrypted, err := s.vault.Decrypt(cred.EncryptedData)
			if err == nil {
				switch cred.Type {
				case "password":
					cfg.Password = decrypted
					cfg.AuthMethod = ssh.AuthPassword
				case "ssh_key", "key":
					cfg.PrivateKey = decrypted
					cfg.AuthMethod = ssh.AuthPublicKey
				}
			}
		}
	}

	// ... rest of logic ...

	// Resolve Jump Hosts
	if host.JumpHostID != "" {
		visited := map[string]bool{host.ID: true}
		s.resolveJumpHosts(ctx, &cfg, host.JumpHostID, 0, visited)
	}

	return &cfg
}

// resolveJumpHosts recursively populates the JumpHosts slice in the config.
func (s *SSHService) resolveJumpHosts(ctx context.Context, cfg *ssh.ConnectionConfig, jumpHostID string, depth int, visited map[string]bool) {
	if depth >= 3 {
		s.log.Warn("Maximum jump host depth reached for host: %s", cfg.Host)
		return
	}

	// SEC-32: Cycle detection
	if visited[jumpHostID] {
		s.log.Warn("Jump host cycle detected at host ID: %s — aborting chain", jumpHostID)
		return
	}
	visited[jumpHostID] = true

	jumpHost, err := s.hosts.GetByID(ctx, jumpHostID)
	if err != nil {
		s.log.Error("Failed to fetch jump host %s: %v", jumpHostID, err)
		return
	}

	// Prepare jump host config
	jumpCfg := ssh.JumpHostConfig{
		Host:     jumpHost.Hostname,
		Port:     jumpHost.Port,
		Username: jumpHost.Username,
	}

	// SECURITY: fetch and decrypt jump host password at connect time
	if jumpHost.HasPassword && s.vault != nil && s.vault.IsUnlocked() {
		if pw, err := s.fetchAndDecryptHostPassword(ctx, jumpHost.ID); err == nil {
			jumpCfg.Password = pw
		}
	}

	if jumpCfg.Port == 0 {
		jumpCfg.Port = 22
	}

	// Resolve Credentials for Jump Host
	if jumpHost.CredentialID != "" && s.vault.IsUnlocked() {
		cred, err := s.creds.GetByID(ctx, jumpHost.CredentialID)
		if err == nil {
			decrypted, err := s.vault.Decrypt(cred.EncryptedData)
			if err == nil {
				switch cred.Type {
				case "password":
					jumpCfg.Password = decrypted
					jumpCfg.AuthMethod = ssh.AuthPassword
				case "ssh_key", "key":
					jumpCfg.PrivateKey = decrypted
					jumpCfg.AuthMethod = ssh.AuthPublicKey
				}
			}
		}
	}
	// ...
	cfg.JumpHosts = append([]ssh.JumpHostConfig{jumpCfg}, cfg.JumpHosts...)

	if jumpHost.JumpHostID != "" {
		s.resolveJumpHosts(ctx, cfg, jumpHost.JumpHostID, depth+1, visited)
	}
}

func (s *SSHService) registerSessionCallbacks(ctx context.Context, session *ssh.Session, host *database.Host) {
	sessionID := session.ID

	// Initialize batcher for this session
	s.batchers.Store(sessionID, NewOutputBatcher(ctx, sessionID))

	session.SetCallbacks(
		func(sessionID string, data []byte) {
			if b, ok := s.batchers.Load(sessionID); ok {
				batcher := b.(*OutputBatcher)
				_, _ = batcher.Write(data)
			} else {
				// Fallback if batcher somehow missing
				encoded := base64.StdEncoding.EncodeToString(data)
				EmitEvent(fmt.Sprintf("session.output.%s", sessionID), encoded)
			}

			if s.onOutputCb != nil {
				s.onOutputCb(sessionID, host.Hostname, string(data))
			}

			if s.shareManager != nil {
				s.shareManager.BroadcastData(sessionID, data)
			}
			if s.recordingManager != nil {
				s.recordingManager.RecordOutput(sessionID, data)
			}
			if s.tailingSvc != nil {
				s.tailingSvc.RegisterOutput(sessionID, host.Label, string(data))
			}
		},
		func(sessionID string) {
			s.log.Info("Session closed: %s", sessionID)

			// Cleanup batcher
			if b, ok := s.batchers.LoadAndDelete(sessionID); ok {
				b.(*OutputBatcher).Flush()
			}

			s.manager.Remove(sessionID)
			s.bus.Publish(eventbus.EventConnectionClosed, sessionID)
		},
		func(sessionID string, err error) {
			s.log.Error("Session error [%s]: %v", sessionID, err)
			s.bus.Publish(eventbus.EventConnectionError, err)
			EmitEvent("session.error", map[string]string{
				"sessionId": sessionID,
				"error":     err.Error(),
			})
		},
	)
}

// ... rest of methods ...

// PushCredential fetches a secret from the vault and injects it into the session's stdin
func (s *SSHService) PushCredential(ctx context.Context, sessionID string, credentialID string) error {
	s.log.Info("Tactical Injection: Pushing credential %s to session %s", credentialID, sessionID)

	if s.vault != nil && !s.vault.IsUnlocked() {
		return fmt.Errorf("vault is locked")
	}

	cred, err := s.creds.GetByID(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("fetch credential: %w", err)
	}

	decrypted, err := s.vault.Decrypt(cred.EncryptedData)
	if err != nil {
		return fmt.Errorf("decrypt failed: %w", err)
	}

	// Ensure memory is aggressively wiped even if a panic occurs
	defer vault.ZeroSlice(decrypted)

	session, ok := s.manager.Get(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Auto-append newline for password/token injection (common use case)
	payload := append(decrypted, '\n')
	defer vault.ZeroSlice(payload) // Wipe the payload slice too

	if s.recordingManager != nil {
		// Log the injection event but NOT the secret itself
		s.recordingManager.RecordInput(sessionID, []byte(fmt.Sprintf("[VAULT INJECTION: %s]", cred.Label)))
	}

	return session.Write(payload)
}



// Exec runs a non-interactive command on an active session and returns output
func (s *SSHService) Exec(sessionID string, cmd string) (string, error) {
	if !s.sanitizer.IsSafe(cmd) {
		s.log.Warn("[SECURITY] Blocked dangerous command execution attempt: %s", cmd)
		return "", fmt.Errorf("command violates security policy")
	}

	session, ok := s.manager.Get(sessionID)
	if !ok {
		return "", fmt.Errorf("session %s not found", sessionID)
	}
	output, err := session.GetClient().ExecuteCommand(cmd)
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

// Resize changes the terminal dimensions for a session
func (s *SSHService) Resize(sessionID string, cols int, rows int) error {
	session, ok := s.manager.Get(sessionID)
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return session.Resize(cols, rows)
}

// GetActiveSessions returns all active session IDs
func (s *SSHService) GetActiveSessions() []map[string]interface{} {
	sessions := s.manager.GetAll()
	result := make([]map[string]interface{}, 0, len(sessions))
	for _, sess := range sessions {
		bytesIn, bytesOut, uptime := sess.Metrics()
		result = append(result, map[string]interface{}{
			"id":        sess.ID,
			"hostId":    sess.HostID,
			"hostLabel": sess.HostLabel,
			"status":    string(sess.Status),
			"startedAt": sess.StartedAt,
			"bytesIn":   bytesIn,
			"bytesOut":  bytesOut,
			"uptime":    uptime.Seconds(),
		})
	}
	return result
}

// CloseAll closes all SSH sessions
func (s *SSHService) CloseAll() {
	if s.log != nil {
		s.log.Info("Closing all SSH sessions")
	}

	// Flush all batchers and clear the map
	s.batchers.Range(func(key, value interface{}) bool {
		value.(*OutputBatcher).Flush()
		s.batchers.Delete(key)
		return true
	})

	if s.manager != nil {
		s.manager.CloseAll()
	}
}

// ImportSSHConfig imports hosts from ~/.ssh/config
func (s *SSHService) ImportSSHConfig() ([]ssh.SSHConfigEntry, error) {
	return ssh.ParseSSHConfig()
}

// DeployKey generates a local SSH key and deploys it to the remote host
func (s *SSHService) DeployKey(hostID string, password string) error {
	if s.vault != nil && !s.vault.IsUnlocked() {
		return fmt.Errorf("vault is locked")
	}
	s.log.Info("Deploying SSH key to host: %s", hostID)

	host, err := s.hosts.GetByID(s.ctx, hostID)
	if err != nil {
		return err
	}

	// 1. Get or generate key
	home, _ := os.UserHomeDir()
	appDir := filepath.Join(home, ".oblivrashell")
	os.MkdirAll(appDir, 0700)
	keyPath := filepath.Join(appDir, "id_ed25519")

	var pubKey []byte
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		priv, pub, err := ssh.GenerateED25519Keypair()
		if err != nil {
			return err
		}
		os.WriteFile(keyPath, priv, 0600)
		os.WriteFile(keyPath+".pub", pub, 0644)
		pubKey = pub
	} else {
		pubKey, _ = os.ReadFile(keyPath + ".pub")
	}

	// 2. Connect via password
	sshConfig := ssh.DefaultConfig()
	sshConfig.Host = host.Hostname
	sshConfig.Port = host.Port
	sshConfig.Username = host.Username
	sshConfig.Password = []byte(password)
	sshConfig.AuthMethod = ssh.AuthPassword

	client := ssh.NewClient(sshConfig)
	err = client.Connect()
	if err != nil {
		return fmt.Errorf("connect with password: %w", err)
	}
	defer client.Close()

	// 3. Deploy key via SFTP to avoid any shell injection risk with the public key bytes
	// First ensure .ssh directory exists
	_, _ = client.ExecuteCommand("mkdir -p ~/.ssh && chmod 700 ~/.ssh")

	sftpClient, err := client.SftpClient()
	if err != nil {
		// SEC-22: Fallback uses stdin pipe to avoid shell interpolation and process listing exposure
		// Write the raw public key bytes directly via a heredoc-style stdin approach
		pubKeyStr := strings.TrimSpace(string(pubKey))
		cmd := "cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
		_, err = client.ExecuteCommandWithStdin(cmd, pubKeyStr+"\n")
		if err != nil {
			return fmt.Errorf("deploy key command: %w", err)
		}
	} else {
		defer sftpClient.Close()
		// Open or create authorized_keys with append flag
		f, err := sftpClient.OpenFile(".ssh/authorized_keys", os.O_WRONLY|os.O_APPEND|os.O_CREATE)
		if err != nil {
			return fmt.Errorf("open authorized_keys via sftp: %w", err)
		}
		defer f.Close()
		entry := append(pubKey, '\n')
		if _, err := f.Write(entry); err != nil {
			return fmt.Errorf("write authorized_keys via sftp: %w", err)
		}
		_, _ = client.ExecuteCommand("chmod 600 ~/.ssh/authorized_keys")
	}

	host.AuthMethod = "key"
	return s.hosts.Update(s.ctx, host)
}

// ListDirectory returns the contents of a directory on the remote host via SFTP
func (s *SSHService) ListDirectory(ctxID string, path string) ([]FileInfo, error) {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return nil, fmt.Errorf("session %s not found", ctxID)
	}

	sc, err := session.GetSftpClient()
	if err != nil {
		return nil, err
	}

	if path == "" || path == "~" {
		path, err = sc.Getwd()
		if err != nil {
			path = "."
		}
	}

	files, err := sc.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", path, err)
	}

	var result []FileInfo
	for _, f := range files {
		result = append(result, FileInfo{
			Name:    f.Name(),
			Size:    f.Size(),
			Mode:    f.Mode().String(),
			ModTime: f.ModTime().Format(time.RFC3339),
			IsDir:   f.IsDir(),
		})
	}
	return result, nil
}

// ReadFile downloads a file from the remote host and returns its base64 encoded content
func (s *SSHService) ReadFile(ctxID string, path string) (string, error) {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return "", fmt.Errorf("session %s not found", ctxID)
	}

	sc, err := session.GetSftpClient()
	if err != nil {
		return "", err
	}

	file, err := sc.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > 5*1024*1024 {
		return "", fmt.Errorf("file too large (%d bytes), maximum allowed for preview is 5MB. Use SFTP download instead", info.Size())
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(content), nil
}

// WriteFile uploads a base64 encoded string to the remote file path
func (s *SSHService) WriteFile(ctxID string, path string, contentBase64 string) error {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return fmt.Errorf("session %s not found", ctxID)
	}

	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return fmt.Errorf("invalid base64 content: %w", err)
	}

	sc, err := session.GetSftpClient()
	if err != nil {
		return err
	}

	file, err := sc.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(content)
	return err
}

// Mkdir creates a new directory on the remote host
func (s *SSHService) Mkdir(ctxID string, path string) error {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return fmt.Errorf("session %s not found", ctxID)
	}
	sc, err := session.GetSftpClient()
	if err != nil {
		return err
	}
	return sc.Mkdir(path)
}

// Rename renames or moves a file/directory on the remote host
func (s *SSHService) Rename(ctxID string, oldPath, newPath string) error {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return fmt.Errorf("session %s not found", ctxID)
	}
	sc, err := session.GetSftpClient()
	if err != nil {
		return err
	}
	return sc.Rename(oldPath, newPath)
}

// Remove deletes a file or directory (must be empty) on the remote host
func (s *SSHService) Remove(ctxID string, path string) error {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return fmt.Errorf("session %s not found", ctxID)
	}
	sc, err := session.GetSftpClient()
	if err != nil {
		return err
	}

	stat, err := sc.Stat(path)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return sc.RemoveDirectory(path)
	}
	return sc.Remove(path)
}

// DirectoryDiff holds the comparison result between a local and remote path
type DirectoryDiff struct {
	Path  string     `json:"path"`
	Items []DiffItem `json:"items"`
}

// DiffItem represents a single file or directory difference
type DiffItem struct {
	Name          string `json:"name"`
	IsDir         bool   `json:"is_dir"`
	Status        string `json:"status"` // 'missing_local', 'missing_remote', 'modified', 'identical'
	LocalSize     int64  `json:"local_size"`
	RemoteSize    int64  `json:"remote_size"`
	LocalModTime  int64  `json:"local_mod_time"`  // unix timestamp
	RemoteModTime int64  `json:"remote_mod_time"` // unix timestamp
}

// CompareDirectories compares a local and remote directory and returns the differences
func (s *SSHService) CompareDirectories(ctxID string, localPath string, remotePath string) (*DirectoryDiff, error) {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return nil, fmt.Errorf("session %s not found", ctxID)
	}

	sc, err := session.GetSftpClient()
	if err != nil {
		return nil, err
	}

	// 1. Read Remote Directory
	remoteItems, err := sc.ReadDir(remotePath)
	if err != nil {
		// If remote doesn't exist, all local files are 'missing_remote'
		remoteItems = []os.FileInfo{}
	}

	remoteMap := make(map[string]os.FileInfo)
	for _, fi := range remoteItems {
		remoteMap[fi.Name()] = fi
	}

	// 2. Read Local Directory
	localItems, err := os.ReadDir(localPath)
	if err != nil {
		// If local doesn't exist, all remote files are 'missing_local'
		localItems = []os.DirEntry{}
	}

	localMap := make(map[string]os.FileInfo)
	for _, de := range localItems {
		info, err := de.Info()
		if err == nil {
			localMap[de.Name()] = info
		}
	}

	// 3. Compare Items
	var diffs []DiffItem

	// Check local items against remote
	for name, localInfo := range localMap {
		remoteInfo, exists := remoteMap[name]

		item := DiffItem{
			Name:         name,
			IsDir:        localInfo.IsDir(),
			LocalSize:    localInfo.Size(),
			LocalModTime: localInfo.ModTime().Unix(),
		}

		if !exists {
			item.Status = "missing_remote"
		} else {
			item.RemoteSize = remoteInfo.Size()
			item.RemoteModTime = remoteInfo.ModTime().Unix()

			if localInfo.IsDir() != remoteInfo.IsDir() || localInfo.Size() != remoteInfo.Size() {
				item.Status = "modified"
			} else {
				item.Status = "identical"
			}
			// Mark as processed
			delete(remoteMap, name)
		}
		diffs = append(diffs, item)
	}

	// Remaining remote items are missing locally
	for name, remoteInfo := range remoteMap {
		diffs = append(diffs, DiffItem{
			Name:          name,
			IsDir:         remoteInfo.IsDir(),
			Status:        "missing_local",
			RemoteSize:    remoteInfo.Size(),
			RemoteModTime: remoteInfo.ModTime().Unix(),
		})
	}

	// Sort diffs: directories first, then alphabetical
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].IsDir != diffs[j].IsDir {
			return diffs[i].IsDir
		}
		return diffs[i].Name < diffs[j].Name
	})

	return &DirectoryDiff{
		Path:  remotePath,
		Items: diffs,
	}, nil
}

// SftpChmod changes the permissions of a file on the remote host
func (s *SSHService) SftpChmod(ctxID string, path string, mode uint32) error {
	session, ok := s.manager.Get(ctxID)
	if !ok {
		return fmt.Errorf("session %s not found", ctxID)
	}
	sc, err := session.GetSftpClient()
	if err != nil {
		return err
	}
	return sc.Chmod(path, os.FileMode(mode))
}

// SftpDownloadAsync queues a file transfer to background worker
func (s *SSHService) SftpDownloadAsync(sessionID, remotePath, localPath string, fileSize int64) (string, error) {
	if s.transferManager == nil {
		return "", fmt.Errorf("transfer manager not initialized")
	}
	return s.transferManager.SftpDownloadAsync(sessionID, remotePath, localPath, fileSize)
}

// SftpUploadAsync queues a file transfer to background worker
func (s *SSHService) SftpUploadAsync(sessionID, localPath, remotePath string) (string, error) {
	if s.transferManager == nil {
		return "", fmt.Errorf("transfer manager not initialized")
	}
	return s.transferManager.SftpUploadAsync(sessionID, localPath, remotePath)
}

// GetTransferState returns all active/recent transfers
func (s *SSHService) GetTransferState() []TransferJob {
	if s.transferManager == nil {
		return []TransferJob{}
	}
	return s.transferManager.GetTransferState()
}

// CancelTransfer aborts a background sync
func (s *SSHService) CancelTransfer(jobID string) error {
	if s.transferManager == nil {
		return fmt.Errorf("transfer manager not initialized")
	}
	return s.transferManager.CancelTransfer(jobID)
}

// ClearTransfers cleans up the transfer queue
func (s *SSHService) ClearTransfers() {
	if s.transferManager != nil {
		s.transferManager.ClearTransfers()
	}
}


// Dependencies returns service dependencies.
// Note: eventbus is infrastructure wired at construction time, not a kernel-managed service.
func (s *SSHService) Dependencies() []string {
	return []string{"vault"}
}

// GetActiveSessionForHost returns the first active SSH session ID for a given hostID.
// Implements isolation.HostSessionResolver so NetworkIsolator can execute remote commands.
func (s *SSHService) GetActiveSessionForHost(hostID string) (string, bool) {
	if s.manager == nil {
		return "", false
	}
	all := s.manager.GetAll()
	for _, sess := range all {
		if sess.HostID == hostID && sess.Status == ssh.SessionActive {
			return sess.ID, true
		}
	}
	return "", false
}
