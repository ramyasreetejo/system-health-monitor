package internal

import "sync"

// ServiceStore is a thread-safe in-memory store for services
type ServiceStore struct {
	mu       sync.RWMutex
	services map[string]*Service
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{
		services: make(map[string]*Service),
	}
}

// Register adds or replaces a service
func (s *ServiceStore) Register(svc *Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[svc.ID] = svc
}

// List returns a snapshot slice of services
func (s *ServiceStore) List() []*Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Service, 0, len(s.services))
	for _, svc := range s.services {
		out = append(out, svc)
	}
	return out
}

// Update replaces the stored service (used by monitor to persist metrics)
func (s *ServiceStore) Update(svc *Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// store the pointer (svc should be a copy for safety if callers want)
	s.services[svc.ID] = svc
}
