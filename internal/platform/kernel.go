package platform

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Kernel acts as the central orchestrator for the platform's runtime.
// It manages service startup in dependency order and shutdown in reverse.
type Kernel struct {
	registry *Registry
	order    []Service
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewKernel creates a new platform kernel with resolved service ordering.
func NewKernel(reg *Registry) (*Kernel, error) {
	order, err := reg.ResolveOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve service order: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Kernel{
		registry: reg,
		order:    order,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Start initiates all registered services in topological order.
func (k *Kernel) Start() error {
	for _, svc := range k.order {
		log.Printf("[KERNEL] Starting service: %s", svc.Name())
		if err := k.safeStart(svc); err != nil {
			return fmt.Errorf("failed to start %s: %w", svc.Name(), err)
		}
	}
	log.Printf("[KERNEL] All services started successfully")
	return nil
}

// Stop gracefully shuts down all services in reverse startup order.
func (k *Kernel) Stop() {
	log.Printf("[KERNEL] Initiating graceful shutdown...")
	k.cancel()

	// Stop in reverse order
	for i := len(k.order) - 1; i >= 0; i-- {
		svc := k.order[i]
		log.Printf("[KERNEL] Stopping service: %s", svc.Name())
		if err := svc.Stop(context.Background()); err != nil {
			log.Printf("[KERNEL] ERROR: Failed to stop %s: %v", svc.Name(), err)
		}
	}
	log.Printf("[KERNEL] Shutdown complete")
}

// Wait blocks until an OS interrupt signal is received, then triggers shutdown.
func (k *Kernel) Wait() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	s := <-sig
	log.Printf("[KERNEL] Signal received: %v", s)
	k.Stop()
}

// safeStart wraps service startup with panic recovery.
func (k *Kernel) safeStart(s Service) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in service %s: %v", s.Name(), r)
		}
	}()
	return s.Start(k.ctx)
}

// Context returns the kernel's base context.
func (k *Kernel) Context() context.Context {
	return k.ctx
}
