package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/ssh"
)

// SessionProvider defines the subset of SSHService needed for transfers
type SessionProvider interface {
	GetSession(id string) (*ssh.Session, bool)
	Mkdir(ctxID string, path string) error
}

// TransferManager handles asynchronous file operations
type TransferManager struct {
	mu              *sync.RWMutex
	jobs            map[string]*TransferJob
	sessionProvider SessionProvider
	bus             *eventbus.Bus
	log             *logger.Logger
}

// NewTransferManager creates a new background worker for SFTP
func NewTransferManager(sp SessionProvider, bus *eventbus.Bus, log *logger.Logger) *TransferManager {
	return &TransferManager{
		jobs:            make(map[string]*TransferJob),
		sessionProvider: sp,
		bus:             bus,
		log:             log.WithPrefix("transfer_manager"),
		mu:              &sync.RWMutex{},
	}
}

// SftpDownloadAsync starts a new background download job
func (m *TransferManager) SftpDownloadAsync(sessionID, remotePath, localPath string, fileSize int64) (string, error) {
	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())

	job := &TransferJob{
		ID:         id,
		SessionID:  sessionID,
		Type:       TransferDownload,
		RemotePath: remotePath,
		LocalPath:  localPath,
		Filename:   filepath.Base(remotePath),
		TotalBytes: fileSize,
		Status:     StatusQueued,
		StartedAt:  time.Now().Format(time.RFC3339),
		cancelFn:   cancel,
		ctx:        ctx,
	}

	m.mu.Lock()
	m.jobs[id] = job
	m.mu.Unlock()

	go m.executeDownload(job)

	m.emitState()
	return id, nil
}

// SftpUploadAsync starts a new background upload job
func (m *TransferManager) SftpUploadAsync(sessionID, localPath, remotePath string) (string, error) {
	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())

	stat, err := os.Stat(localPath)
	if err != nil {
		cancel()
		return "", fmt.Errorf("stat local file: %w", err)
	}

	job := &TransferJob{
		ID:         id,
		SessionID:  sessionID,
		Type:       TransferUpload,
		RemotePath: remotePath,
		LocalPath:  localPath,
		Filename:   filepath.Base(localPath),
		TotalBytes: stat.Size(),
		Status:     StatusQueued,
		StartedAt:  time.Now().Format(time.RFC3339),
		cancelFn:   cancel,
		ctx:        ctx,
	}

	m.mu.Lock()
	m.jobs[id] = job
	m.mu.Unlock()

	go m.executeUpload(job)

	m.emitState()
	return id, nil
}

// CancelTransfer gracefully aborts an active transfer
func (m *TransferManager) CancelTransfer(id string) error {
	m.mu.RLock()
	job, ok := m.jobs[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Status == StatusInProgress || job.Status == StatusQueued {
		job.cancelFn() // Trigger context cancellation
		m.updateJobStatus(id, StatusCancelled, nil)
	}

	return nil
}

// GetTransferState returns all current and recent transfers
func (m *TransferManager) GetTransferState() []TransferJob {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []TransferJob
	for _, job := range m.jobs {
		active = append(active, *job)
	}
	return active
}

func (m *TransferManager) executeDownload(job *TransferJob) {
	defer job.cancelFn()
	m.updateJobStatus(job.ID, StatusInProgress, nil)

	session, ok := m.sessionProvider.GetSession(job.SessionID)
	if !ok {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("session disconnected"))
		return
	}

	sc, err := session.GetSftpClient()
	if err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("sftp error: %v", err))
		return
	}

	remoteFile, err := sc.Open(job.RemotePath)
	if err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("open remote file: %v", err))
		return
	}
	defer remoteFile.Close()

	if job.TotalBytes <= 0 {
		if stat, err := remoteFile.Stat(); err == nil {
			m.mu.Lock()
			job.TotalBytes = stat.Size()
			m.mu.Unlock()
		}
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(job.LocalPath), 0755); err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("create local dir: %v", err))
		return
	}

	localFile, err := os.Create(job.LocalPath)
	if err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("create local file: %v", err))
		return
	}
	defer localFile.Close()

	progressWriter := &progressTracker{
		jobID:   job.ID,
		manager: m,
		ctx:     job.ctx,
	}

	_, err = io.Copy(localFile, io.TeeReader(remoteFile, progressWriter))

	if err != nil {
		if err == context.Canceled {
			m.log.Info("Download cancelled: %s", job.ID)
			os.Remove(job.LocalPath)
			return
		}
		m.updateJobStatus(job.ID, StatusFailed, err)
		return
	}

	m.updateJobStatus(job.ID, StatusCompleted, nil)
}

func (m *TransferManager) executeUpload(job *TransferJob) {
	defer job.cancelFn()
	m.updateJobStatus(job.ID, StatusInProgress, nil)

	session, ok := m.sessionProvider.GetSession(job.SessionID)
	if !ok {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("session disconnected"))
		return
	}

	sc, err := session.GetSftpClient()
	if err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("sftp error: %v", err))
		return
	}

	localFile, err := os.Open(job.LocalPath)
	if err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("open local file: %v", err))
		return
	}
	defer localFile.Close()

	_ = m.sessionProvider.Mkdir(job.SessionID, filepath.Dir(filepath.ToSlash(job.RemotePath)))

	remoteFile, err := sc.Create(job.RemotePath)
	if err != nil {
		m.updateJobStatus(job.ID, StatusFailed, fmt.Errorf("create remote file: %v", err))
		return
	}
	defer remoteFile.Close()

	progressWriter := &progressTracker{
		jobID:   job.ID,
		manager: m,
		ctx:     job.ctx,
	}

	_, err = io.Copy(remoteFile, io.TeeReader(localFile, progressWriter))

	if err != nil {
		if err == context.Canceled {
			m.log.Info("Upload cancelled: %s", job.ID)
			sc.Remove(job.RemotePath)
			return
		}
		m.updateJobStatus(job.ID, StatusFailed, err)
		return
	}

	m.updateJobStatus(job.ID, StatusCompleted, nil)
}

func (m *TransferManager) updateJobStatus(id string, status TransferStatus, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return
	}

	job.Status = status
	if err != nil {
		job.Error = err.Error()
	}

	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		nowStr := time.Now().Format(time.RFC3339)
		job.CompletedAt = &nowStr
	}

	m.bus.Publish("sftp_transfer_update", *job)
}

func (m *TransferManager) emitState() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var active []TransferJob
	for _, job := range m.jobs {
		active = append(active, *job)
	}
	m.bus.Publish("sftp_transfers_list", active)
}

type progressTracker struct {
	jobID       string
	manager     *TransferManager
	ctx         context.Context
	lastUpdated time.Time
	bytesSince  int64
}

func (pt *progressTracker) Write(p []byte) (n int, err error) {
	select {
	case <-pt.ctx.Done():
		return 0, context.Canceled
	default:
	}

	n = len(p)
	now := time.Now()

	pt.manager.mu.Lock()
	job, ok := pt.manager.jobs[pt.jobID]
	if !ok {
		pt.manager.mu.Unlock()
		return n, nil
	}

	job.BytesCopied += int64(n)
	pt.bytesSince += int64(n)

	if now.Sub(pt.lastUpdated) > 500*time.Millisecond {
		durationSec := now.Sub(pt.lastUpdated).Seconds()
		if durationSec > 0 {
			job.SpeedBytesS = float64(pt.bytesSince) / durationSec
		}

		pt.lastUpdated = now
		pt.bytesSince = 0

		jobCopy := *job
		pt.manager.mu.Unlock()

		pt.manager.bus.Publish("sftp_transfer_update", jobCopy)
	} else {
		pt.manager.mu.Unlock()
	}

	return n, nil
}

// ClearTransfers removes finished/failed/cancelled jobs from the queue
func (m *TransferManager) ClearTransfers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, job := range m.jobs {
		if job.Status == StatusCompleted || job.Status == StatusFailed || job.Status == StatusCancelled {
			delete(m.jobs, id)
		}
	}
}
