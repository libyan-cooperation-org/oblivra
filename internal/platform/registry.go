package platform

import (
	"fmt"
)

// Registry manages the collection of services and determines their startup order.
type Registry struct {
	services map[string]Service
}

// NewRegistry creates a new service registry.
func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]Service),
	}
}

// Register adds a service to the registry.
func (r *Registry) Register(s Service) error {
	name := s.Name()
	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service already registered: %s", name)
	}
	r.services[name] = s
	return nil
}

// GetServices returns all registered services.
func (r *Registry) GetServices() map[string]Service {
	return r.services
}

// ResolveOrder performs a topological sort to determine the correct startup order based on dependencies.
func (r *Registry) ResolveOrder() ([]Service, error) {
	visited := make(map[string]bool)
	stack := make(map[string]bool)
	order := []Service{}

	var visit func(string) error
	visit = func(name string) error {
		if stack[name] {
			return fmt.Errorf("circular dependency detected involving service: %s", name)
		}
		if visited[name] {
			return nil
		}

		svc, exists := r.services[name]
		if !exists {
			return fmt.Errorf("missing dependency: %s", name)
		}

		stack[name] = true
		for _, dep := range svc.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}
		stack[name] = false
		visited[name] = true

		order = append(order, svc)
		return nil
	}

	for name := range r.services {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}
