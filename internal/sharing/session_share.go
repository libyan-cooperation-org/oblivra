package sharing

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	vaultpkg "github.com/kingknull/oblivrashell/internal/vault"
)

// ShareMode defines the access level
type ShareMode string

const (
	ShareReadOnly  ShareMode = "read_only"
	ShareReadWrite ShareMode = "read_write"
	ShareObserve   ShareMode = "observe" // can see output but no interaction
)

// SessionShare represents a shared session link
type SessionShare struct {
	ID             string             `json:"id"`
	SessionID      string             `json:"session_id"`
	HostLabel      string             `json:"host_label"`
	Mode           ShareMode          `json:"mode"`
	CreatedBy      string             `json:"created_by"`
	CreatedAt      time.Time          `json:"created_at"`
	ExpiresAt      time.Time          `json:"expires_at"`
	MaxViewers     int                `json:"max_viewers"`
	CurrentViewers int                `json:"current_viewers"`
	AccessToken    string             `json:"access_token"` // hashed, never stored raw
	EncryptionKey  string             `json:"-"`            // only returned at creation
	IsActive       bool               `json:"is_active"`
	AccessLog      []ShareAccessEntry `json:"access_log"`
}

// ShareAccessEntry records who accessed a shared session
type ShareAccessEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	ViewerID   string    `json:"viewer_id"`
	ViewerName string    `json:"viewer_name"`
	Action     string    `json:"action"` // "joined", "left", "input_attempt"
	IPAddress  string    `json:"ip_address,omitempty"`
}

// ShareBuffer holds the terminal output buffer for sharing
type ShareBuffer struct {
	mu       sync.RWMutex
	data     [][]byte
	maxSize  int
	totalLen int
}

func NewShareBuffer(maxSize int) *ShareBuffer {
	return &ShareBuffer{
		data:    make([][]byte, 0, 1024),
		maxSize: maxSize,
	}
}

// Write appends data to the buffer
func (b *ShareBuffer) Write(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	chunk := make([]byte, len(data))
	copy(chunk, data)
	b.data = append(b.data, chunk)
	b.totalLen += len(data)

	// Evict oldest if over max size
	for b.totalLen > b.maxSize && len(b.data) > 1 {
		b.totalLen -= len(b.data[0])
		b.data = b.data[1:]
	}
}

// Snapshot returns a copy of the entire buffer
func (b *ShareBuffer) Snapshot() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]byte, 0, b.totalLen)
	for _, chunk := range b.data {
		result = append(result, chunk...)
	}
	return result
}

// SinceIndex returns data written after a given index
func (b *ShareBuffer) SinceIndex(index int) ([]byte, int) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if index >= len(b.data) {
		return nil, len(b.data)
	}

	var result []byte
	for i := index; i < len(b.data); i++ {
		result = append(result, b.data[i]...)
	}
	return result, len(b.data)
}

// EventBus interface to avoid cyclic dependency
type EventBus interface {
	Publish(topic string, data interface{})
}

// ShareManager manages all shared sessions
type ShareManager struct {
	mu       sync.RWMutex
	shares   map[string]*SessionShare
	buffers  map[string]*ShareBuffer            // keyed by session ID
	viewers  map[string]map[string]*ShareViewer // share ID -> viewer ID -> viewer
	bus      EventBus                           // For broadcasting joining/leaving
	executor SessionExecutor
}

// ShareViewer represents a connected viewer
type ShareViewer struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	JoinedAt  time.Time   `json:"joined_at"`
	LastPoll  time.Time   `json:"last_poll"`
	BufferIdx int         `json:"-"`
	DataChan  chan []byte `json:"-"`
}

func NewShareManager(executor SessionExecutor, bus EventBus) *ShareManager {
	return &ShareManager{
		shares:   make(map[string]*SessionShare),
		buffers:  make(map[string]*ShareBuffer),
		viewers:  make(map[string]map[string]*ShareViewer),
		executor: executor,
		bus:      bus,
	}
}

// CreateShare creates a new session share
func (m *ShareManager) CreateShare(
	sessionID string,
	hostLabel string,
	mode ShareMode,
	createdBy string,
	expiresIn time.Duration,
	maxViewers int,
) (*SessionShare, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate access token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	rawToken := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash the token for storage
	tokenHash := hashToken(rawToken)

	// Generate encryption key for the share
	encKeyBytes := make([]byte, 32)
	if _, err := rand.Read(encKeyBytes); err != nil {
		return nil, "", fmt.Errorf("generate encryption key: %w", err)
	}
	encKey := hex.EncodeToString(encKeyBytes)

	if maxViewers <= 0 {
		maxViewers = 5
	}

	if expiresIn <= 0 {
		expiresIn = 1 * time.Hour
	}

	share := &SessionShare{
		ID:             uuid.New().String(),
		SessionID:      sessionID,
		HostLabel:      hostLabel,
		Mode:           mode,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(expiresIn),
		MaxViewers:     maxViewers,
		CurrentViewers: 0,
		AccessToken:    tokenHash,
		EncryptionKey:  encKey,
		IsActive:       true,
		AccessLog:      make([]ShareAccessEntry, 0),
	}

	m.shares[share.ID] = share

	// Create buffer if not exists
	if _, ok := m.buffers[sessionID]; !ok {
		m.buffers[sessionID] = NewShareBuffer(5 * 1024 * 1024) // 5MB buffer
	}

	// Initialize viewer map
	m.viewers[share.ID] = make(map[string]*ShareViewer)

	// Build share link
	shareLink := fmt.Sprintf("sovereign://%s?token=%s&key=%s", share.ID, rawToken, encKey)

	return share, shareLink, nil
}

// ValidateAccess checks if a token is valid for a share
func (m *ShareManager) ValidateAccess(shareID string, token string) (*SessionShare, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	share, ok := m.shares[shareID]
	if !ok {
		return nil, fmt.Errorf("share not found")
	}

	if !share.IsActive {
		return nil, fmt.Errorf("share is no longer active")
	}

	if time.Now().After(share.ExpiresAt) {
		return nil, fmt.Errorf("share has expired")
	}

	tokenHash := hashToken(token)
	if !vaultpkg.SecureCompare([]byte(tokenHash), []byte(share.AccessToken)) {
		return nil, fmt.Errorf("invalid access token")
	}

	if share.CurrentViewers >= share.MaxViewers {
		return nil, fmt.Errorf("maximum viewers reached")
	}

	return share, nil
}

// JoinShare adds a viewer to a share
func (m *ShareManager) JoinShare(shareID string, viewerName string) (*ShareViewer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, ok := m.shares[shareID]
	if !ok {
		return nil, fmt.Errorf("share not found")
	}

	viewer := &ShareViewer{
		ID:       uuid.New().String(),
		Name:     viewerName,
		JoinedAt: time.Now(),
		LastPoll: time.Now(),
		DataChan: make(chan []byte, 256),
	}

	m.viewers[shareID][viewer.ID] = viewer
	share.CurrentViewers++

	// Log access
	share.AccessLog = append(share.AccessLog, ShareAccessEntry{
		Timestamp:  time.Now(),
		ViewerID:   viewer.ID,
		ViewerName: viewerName,
		Action:     "joined",
	})

	if m.bus != nil {
		m.bus.Publish("share.viewer_joined", map[string]interface{}{
			"share_id":   shareID,
			"session_id": share.SessionID,
			"viewer_id":  viewer.ID,
			"name":       viewerName,
		})
	}

	return viewer, nil
}

// HandleViewerInput processes input from a viewer and sends it to the session
func (m *ShareManager) HandleViewerInput(shareID string, viewerID string, token string, data string) error {
	share, err := m.ValidateAccess(shareID, token)
	if err != nil {
		return err
	}

	if share.Mode != ShareReadWrite {
		return fmt.Errorf("this share is read-only")
	}

	m.mu.RLock()
	viewers, ok := m.viewers[shareID]
	if !ok {
		m.mu.RUnlock()
		return fmt.Errorf("share not active")
	}
	_, ok = viewers[viewerID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("viewer not joined")
	}

	if m.executor == nil {
		return fmt.Errorf("no session executor configured")
	}

	// Log input attempt
	m.mu.Lock()
	share.AccessLog = append(share.AccessLog, ShareAccessEntry{
		Timestamp: time.Now(),
		ViewerID:  viewerID,
		Action:    "input_attempt",
	})
	m.mu.Unlock()

	return m.executor.SendInput(share.SessionID, data)
}

// LeaveShare removes a viewer
func (m *ShareManager) LeaveShare(shareID string, viewerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if viewers, ok := m.viewers[shareID]; ok {
		if viewer, ok := viewers[viewerID]; ok {
			close(viewer.DataChan)
			delete(viewers, viewerID)
		}
	}

	if share, ok := m.shares[shareID]; ok {
		share.CurrentViewers--
		share.AccessLog = append(share.AccessLog, ShareAccessEntry{
			Timestamp: time.Now(),
			ViewerID:  viewerID,
			Action:    "left",
		})

		if m.bus != nil {
			m.bus.Publish("share.viewer_left", map[string]interface{}{
				"share_id":   shareID,
				"session_id": share.SessionID,
				"viewer_id":  viewerID,
			})
		}
	}
}

// BroadcastData sends terminal output to all viewers of a session
func (m *ShareManager) BroadcastData(sessionID string, data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Write to buffer
	if buf, ok := m.buffers[sessionID]; ok {
		buf.Write(data)
	}

	// Find all shares for this session
	for shareID, share := range m.shares {
		if share.SessionID != sessionID || !share.IsActive {
			continue
		}

		if viewers, ok := m.viewers[shareID]; ok {
			for _, viewer := range viewers {
				select {
				case viewer.DataChan <- data:
				default:
					// Viewer is too slow, skip
				}
			}
		}
	}
}

// GetSnapshot returns the full buffer for a session
func (m *ShareManager) GetSnapshot(sessionID string) []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if buf, ok := m.buffers[sessionID]; ok {
		return buf.Snapshot()
	}
	return nil
}

// RevokeShare deactivates a share
func (m *ShareManager) RevokeShare(shareID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, ok := m.shares[shareID]
	if !ok {
		return fmt.Errorf("share not found")
	}

	share.IsActive = false

	// Disconnect all viewers
	if viewers, ok := m.viewers[shareID]; ok {
		for id, viewer := range viewers {
			close(viewer.DataChan)
			delete(viewers, id)
		}
	}

	return nil
}

// GetActiveShares returns all active shares
func (m *ShareManager) GetActiveShares() []SessionShare {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var shares []SessionShare
	for _, share := range m.shares {
		if share.IsActive && time.Now().Before(share.ExpiresAt) {
			shares = append(shares, *share)
		}
	}
	return shares
}

// GetSharesBySession returns shares for a specific session
func (m *ShareManager) GetSharesBySession(sessionID string) []SessionShare {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var shares []SessionShare
	for _, share := range m.shares {
		if share.SessionID == sessionID {
			shares = append(shares, *share)
		}
	}
	return shares
}

// GetViewers returns current viewers for a share
func (m *ShareManager) GetViewers(shareID string) []ShareViewer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ShareViewer
	if viewers, ok := m.viewers[shareID]; ok {
		for _, v := range viewers {
			result = append(result, ShareViewer{
				ID:       v.ID,
				Name:     v.Name,
				JoinedAt: v.JoinedAt,
				LastPoll: v.LastPoll,
			})
		}
	}
	return result
}

// GetTotalViewersForSession returns sum of viewers across all active shares for a session
func (m *ShareManager) GetTotalViewersForSession(sessionID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for shareID, share := range m.shares {
		if share.SessionID == sessionID && share.IsActive {
			if viewers, ok := m.viewers[shareID]; ok {
				total += len(viewers)
			}
		}
	}
	return total
}

// CleanupExpired removes expired shares
func (m *ShareManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	now := time.Now()

	for id, share := range m.shares {
		if now.After(share.ExpiresAt) {
			share.IsActive = false
			// Disconnect viewers
			if viewers, ok := m.viewers[id]; ok {
				for vid, viewer := range viewers {
					close(viewer.DataChan)
					delete(viewers, vid)
				}
			}
			count++
		}
	}

	return count
}

func hashToken(token string) string {
	// Use SHA-256 for token hashing
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
