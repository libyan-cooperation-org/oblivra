package sync

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

// SyncItem represents a localized piece of structured data
type SyncItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "host", "snippet", "setting"
	Content   string    `json:"content"`
	UpdatedAt string    `json:"updated_at"`
	DeviceID  string    `json:"device_id"`
	Signature string    `json:"signature"`
	Deleted   bool      `json:"deleted"`
}

// EncryptedPayload represents data sent to the cloud relay
type EncryptedPayload struct {
	DeviceID string `json:"device_id"`
	Nonce    string `json:"nonce"`
	Data     string `json:"data"` // base64
}

// Conflict represents a sync collision
type Conflict struct {
	ItemID     string     `json:"item_id"`
	LocalItem  SyncItem   `json:"local_item"`
	RemoteItem SyncItem   `json:"remote_item"`
	DetectedAt string     `json:"detected_at"`
	ResolvedAt *string    `json:"resolved_at,omitempty"`
	Resolution string     `json:"resolution,omitempty"` // "keep_local", "keep_remote", "merge"
}

// SyncEngine manages E2E cloud synchronization
type SyncEngine struct {
	mu            sync.RWMutex
	deviceID      string
	encryptionKey []byte
	syncEndpoint  string
	lastSync      string
	queue         []SyncItem
	conflicts     []Conflict
	httpClient    *http.Client
	isSyncing     bool
}

func NewSyncEngine(deviceID string, password string) *SyncEngine {
	// Derive encryption key using deviceID as salt
	key := argon2.IDKey([]byte(password), []byte(deviceID), 3, 64*1024, 4, 32)

	return &SyncEngine{
		deviceID:      deviceID,
		encryptionKey: key,
		syncEndpoint:  "https://sync.oblivrashell.dev/api/v1/sync",
		queue:         make([]SyncItem, 0),
		conflicts:     make([]Conflict, 0),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QueueUpdate adds an item to the sync queue
func (e *SyncEngine) QueueUpdate(itemType string, content interface{}, isDeleted bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return err
	}

	item := SyncItem{
		ID:        uuid.New().String(), // In reality, use the actual entity ID
		Type:      itemType,
		Content:   string(contentBytes),
		UpdatedAt: time.Now().Format(time.RFC3339),
		DeviceID:  e.deviceID,
		Deleted:   isDeleted,
	}

	e.queue = append(e.queue, item)
	return nil
}

// Sync performs a bi-directional synchronization with the relay
func (e *SyncEngine) Sync() error {
	e.mu.Lock()
	if e.isSyncing {
		e.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	e.isSyncing = true
	localQueue := make([]SyncItem, len(e.queue))
	copy(localQueue, e.queue)
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.isSyncing = false
		e.mu.Unlock()
	}()

	// 1. Encrypt local changes
	if len(localQueue) > 0 {
		payloadBytes, err := json.Marshal(localQueue)
		if err != nil {
			return err
		}

		encrypted, nonce, err := e.encrypt(payloadBytes)
		if err != nil {
			return err
		}

		payload := EncryptedPayload{
			DeviceID: e.deviceID,
			Nonce:    hex.EncodeToString(nonce),
			Data:     hex.EncodeToString(encrypted),
		}

		// 2. Send to cloud (stub)
		_ = payload
		// err = e.pushToCloud(payload)
	}

	// 3. Fetch remote changes (stub)
	/*
		remotePayload, err := e.fetchFromCloud()
		if err == nil && remotePayload != nil {
			remoteBytes, _ := hex.DecodeString(remotePayload.Data)
			nonce, _ := hex.DecodeString(remotePayload.Nonce)

			decrypted, err := e.decrypt(remoteBytes, nonce)
			if err == nil {
				var remoteItems []SyncItem
				json.Unmarshal(decrypted, &remoteItems)
				e.processRemoteItems(remoteItems)
			}
		}
	*/

	e.mu.Lock()
	// Clear queue of items successfully sent (assuming success for stub)
	e.queue = e.queue[len(localQueue):]
	e.lastSync = time.Now().Format(time.RFC3339)
	e.mu.Unlock()

	return nil
}

// ResolveConflict resolves a sync conflict
func (e *SyncEngine) ResolveConflict(conflictItemID string, resolution string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, c := range e.conflicts {
		if c.ItemID == conflictItemID {
			now := time.Now().Format(time.RFC3339)
			e.conflicts[i].ResolvedAt = &now
			e.conflicts[i].Resolution = resolution

			switch resolution {
			case "keep_local":
				// Re-enqueue local version
				e.queue = append(e.queue, c.LocalItem)
			case "keep_remote":
				// Remote already exists, do nothing (or trigger DB update)
			}
			return nil
		}
	}

	return fmt.Errorf("conflict not found")
}

func (e *SyncEngine) encrypt(data []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(e.encryptionKey)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	return ciphertext, nonce, nil
}

func (e *SyncEngine) decrypt(data []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.encryptionKey)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (e *SyncEngine) pushToCloud(payload EncryptedPayload) error {
	data, _ := json.Marshal(payload)
	resp, err := e.httpClient.Post(e.syncEndpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sync pushing failed: %s", resp.Status)
	}

	return nil
}

func (e *SyncEngine) fetchFromCloud() (*EncryptedPayload, error) {
	url := fmt.Sprintf("%s?device_id=%s&since=%d", e.syncEndpoint, e.deviceID, parseTime(e.lastSync).Unix())
	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync fetching failed: %s", resp.Status)
	}

	var payload EncryptedPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}
