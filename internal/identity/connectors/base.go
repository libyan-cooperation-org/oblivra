package connectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/kingknull/oblivrashell/internal/identity"
)

// IdentityConnector defines the interface for external identity providers.
type IdentityConnector interface {
	// FetchUsers retrieves all users from the remote provider.
	FetchUsers(ctx context.Context) ([]*identity.UserResource, error)
	// Verify tests the connection and credentials.
	Verify(ctx context.Context) error
	// Type returns the provider type (e.g., "okta", "ldap").
	Type() string
	// ID returns the unique identifier for this connector instance.
	ID() string
}

var (
	registry = make(map[string]factory)
	regMu    sync.RWMutex
)

type factory func(id string, configJSON string) (IdentityConnector, error)

// Register registers a connector factory for a given type.
func Register(connectorType string, f factory) {
	regMu.Lock()
	defer regMu.Unlock()
	registry[connectorType] = f
}

// Create instantiates a connector of the given type.
func Create(id string, connectorType string, configJSON string) (IdentityConnector, error) {
	regMu.RLock()
	f, ok := registry[connectorType]
	regMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported identity connector type: %s", connectorType)
	}

	return f(id, configJSON)
}
