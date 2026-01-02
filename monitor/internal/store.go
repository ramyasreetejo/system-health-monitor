package internal

import "sync"

type ServiceStore struct {
	mu       sync.RWMutex
	services map[string]*Service
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{
		services: make(map[string]*Service),
	}
}

func (s *ServiceStore) Register(svc *Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[svc.ID] = svc
}

func (s *ServiceStore) List() []*Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Service, 0, len(s.services))
	for _, svc := range s.services {
		out = append(out, svc)
	}
	return out
}

func (s *ServiceStore) Update(svc *Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[svc.ID] = svc
}
