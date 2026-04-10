package services

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/ssh"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service defines a standard interface for application services
type Service interface {
	Name() string
	Dependencies() []string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// BaseService provides a default implementation for optional methods
type BaseService struct{}

func (s *BaseService) Dependencies() []string          { return nil }
func (s *BaseService) Start(ctx context.Context) error { return nil }
func (s *BaseService) Stop(ctx context.Context) error  { return nil }

// EmitEvent safely wraps wails runtime.EventsEmit to avoid test panics
func EmitEvent(ctx context.Context, eventName string, optionalData ...interface{}) {
	if ctx == nil {
		return
	}
	if ctx.Value("test") != nil {
		return
	}
	// Defensively catch Wails panics if given context lacks expected lifecycle flags
	defer func() {
		if r := recover(); r != nil {
			// Do nothing on panic, it's just a test context lacking Wails bindings
		}
	}()
	if app := application.Get(); app != nil {
		app.Event.Emit(eventName, optionalData...)
	}
}

// SessionExecutor defines the interface for interacting with active SSH/Terminal sessions.
type SessionExecutor interface {
	SendInput(sessionID, data string) error
}

// VaultProvider defines the interface for interacting with the secret vault.
type VaultProvider interface {
	IsSetup() bool
	IsUnlocked() bool
	Unlock(password string, hardwareKey []byte, rememberMe bool) error
	UnlockWithKeychain() error
	Lock()
	Setup(password string, yubiKeySerial string) error
	GetYubiKeySerial() string
	AccessMasterKey(fn func(key []byte) error) error
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

// HostManager defines the interface for managing hosts and connections.
type HostManager interface {
	GetAll() ([]database.Host, error)
	GetByID(id string) (*database.Host, error)
	Update(host *database.Host) error
	Create(host database.Host) (*database.Host, error)
	Delete(id string) error
	ToggleFavorite(id string) error
	WakeHost(id string) error
}

// SSHManager defines the interface for SSH session management.
type SSHManager interface {
	Connect(hostID string) (string, error)
	Disconnect(sessionID string) error
	SendInput(sessionID, data string) error
	Exec(sessionID string, cmd string) (string, error)
	Resize(sessionID string, cols, rows int) error
	GetActiveSessions() []map[string]interface{}
	CloseAll()
	FileSystemProvider
}

// TransferProvider defines the interface for asynchronous file transfers.
type TransferProvider interface {
	SftpDownloadAsync(sessionID, remotePath, localPath string, fileSize int64) (string, error)
	SftpUploadAsync(sessionID, localPath, remotePath string) (string, error)
	GetTransferState() []TransferJob
	CancelTransfer(jobID string) error
	ClearTransfers()
}

// FIDO2Provider defines the interface for security key operations.
type FIDO2Provider interface {
	HasCredentials() bool
	BeginRegistration(userID, userName string) (*security.FIDO2Challenge, error)
	ListCredentials() []security.FIDO2Credential
	RemoveCredential(id string) error
}

// YubiKeyProvider defines the interface for hardware key operations.
type YubiKeyProvider interface {
	Detect() ([]security.YubiKeyInfo, error)
	GenerateSSHKey(serial, slot, pin string) (string, error)
	GetSSHPublicKey(serial, slot string) (string, error)
	DeriveVaultKey(serial string, password string) ([]byte, error)
}

// CertificateProvider defines the interface for SSH certificate management.
type CertificateProvider interface {
	ListCertificates() ([]ssh.CertificateInfo, error)
	CheckExpiry(duration time.Duration) ([]ssh.CertificateInfo, error)
}

// SessionOperations defines the interface for session lifecycle management.
type SessionOperations interface {
	Create(sess database.Session) error
	UpdateStatus(id string, status string) error
}

// FileSystemProvider defines the interface for common filesystem operations.
type FileSystemProvider interface {
	ListDirectory(ctxID string, path string) ([]FileInfo, error)
	ReadFile(ctxID string, path string) (string, error)
	WriteFile(ctxID string, path string, contentBase64 string) error
	Mkdir(ctxID string, path string) error
	Rename(ctxID string, oldPath, newPath string) error
	Remove(ctxID string, path string) error
}

// FileManager defines the unified interface for file operations across local and remote systems.
type FileManager interface {
	Service
	FileSystemProvider
	Download(ctxID string, path string, destPath string, size int64) (string, error)
	Upload(ctxID string, localPath string, destPath string) (string, error)
}

// SnippetManager defines the interface for command snippet operations.
type SnippetManager interface {
	Service
	List() ([]database.Snippet, error)
	Get(id string) (database.Snippet, error)
	Create(title, command, description string, tags, variables []string) (database.Snippet, error)
	Update(id, title, command, description string, tags, variables []string) (database.Snippet, error)
	Delete(id string) error
	ExecuteSnippet(snippetID string, sessionID string, variables map[string]string, autoSudo bool) error
}

// MultiExecProvider defines the interface for concurrent command execution.
type MultiExecProvider interface {
	Service
	Execute(command string, hostIDs []string, timeoutSeconds int) (string, error)
	GetJob(jobID string) (*MultiExecJob, error)
	GetRecentJobs(limit int) []*MultiExecJob
}

// AIPrompter defines the interface for AI-assisted operations.
type AIPrompter interface {
	Service
	ExplainError(errorOutput string) (*AIResponse, error)
	GenerateCommand(description string) (*AIResponse, error)
}

// IncidentManager defines the interface for managing security cases.
type IncidentManager interface {
	Service
	ListIncidents(ctx context.Context, status string, owner string, limit int) ([]database.Incident, error)
	GetIncident(ctx context.Context, id string) (*database.Incident, error)
	UpdateIncidentStatus(ctx context.Context, id string, status string, reason string) error
	AssignIncident(ctx context.Context, id string, owner string) error
	GetTimeline(ctx context.Context, incidentID string) ([]database.AuditLog, error)
	GetEvidence(ctx context.Context, incidentID string) ([]database.EvidenceItem, error)

	// Detection-related methods for AlertingService
	GetByRuleAndGroup(ctx context.Context, ruleID string, groupKey string) (*database.Incident, error)
	Upsert(ctx context.Context, incident *database.Incident) error
	Search(ctx context.Context, status string, owner string, limit int) ([]database.Incident, error)
	UpdateStatus(ctx context.Context, id string, status string, reason string) error
}

// PlaybookProvider defines the interface for automated response chains.
type PlaybookProvider interface {
	Service
	ExecuteAction(ctx context.Context, action string, params map[string]interface{}) (string, error)
	RunPlaybook(ctx context.Context, playbookID string, incidentID string) error
	ListAvailableActions() []string
}
