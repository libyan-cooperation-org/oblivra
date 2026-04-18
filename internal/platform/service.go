package platform

import "context"

// Service defines the standard interface for all OBLIVRA system components.
// Every subsystem (Vault, Auth, Pipeline, API, etc.) must implement this
// to be managed by the Platform Kernel.
type Service interface {
	// Name returns the unique identifier for the service.
	Name() string

	// Dependencies returns a list of service names that must be started before this service.
	Dependencies() []string

	// Start initializes the service and begins its execution.
	// The provided context is used to manage the service's lifecycle.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service.
	Stop(ctx context.Context) error
}

// HealthReporter is an optional interface that allows a Service to report its internal health.
// Services implementing this will automatically be queried by the HealthService.
type HealthReporter interface {
	Health(ctx context.Context) error
}

