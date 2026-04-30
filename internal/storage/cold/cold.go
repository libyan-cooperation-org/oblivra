// Package cold defines the cold-tier abstraction that warm-tier files migrate
// to once they age past the warm threshold. The interface is intentionally
// minimal — Put/Get/List/Delete plus a metadata struct — so a local-disk
// archive (default) and an S3-compatible bucket (build-tag `coldS3`) can both
// implement it without a vendor SDK in the air-gapped binary.
package cold

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ObjectInfo is what List returns per archived file.
type ObjectInfo struct {
	Key      string    `json:"key"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// ObjectStore is the cold-tier sink interface.
type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Backend() string // "local" / "s3" / etc.
}

// LocalStore writes objects under a base directory. Suitable for air-gap
// deployments and testing. WORM-locks every Put for tamper evidence.
type LocalStore struct {
	base string
	mu   sync.Mutex
}

func NewLocalStore(base string) (*LocalStore, error) {
	if base == "" {
		return nil, fmt.Errorf("cold: base dir required")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return &LocalStore{base: base}, nil
}

func (l *LocalStore) Backend() string { return "local" }

func (l *LocalStore) Put(_ context.Context, key string, body io.Reader) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	full := filepath.Join(l.base, key)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	tmp := full + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, full); err != nil {
		return err
	}
	// Make the file read-only to mimic cold-tier WORM semantics.
	_ = os.Chmod(full, 0o444)
	return nil
}

func (l *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	full := filepath.Join(l.base, key)
	return os.Open(full)
}

func (l *LocalStore) List(_ context.Context, prefix string) ([]ObjectInfo, error) {
	out := []ObjectInfo{}
	root := filepath.Join(l.base, prefix)
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(l.base, p)
		out = append(out, ObjectInfo{
			Key: filepath.ToSlash(rel), Size: info.Size(), Modified: info.ModTime(),
		})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

func (l *LocalStore) Delete(_ context.Context, key string) error {
	full := filepath.Join(l.base, key)
	// Strip the WORM bit before deletion so the operator can see this is the
	// canonical path that records every deletion (caller is expected to also
	// add an audit-chain entry).
	_ = os.Chmod(full, 0o600)
	return os.Remove(full)
}
