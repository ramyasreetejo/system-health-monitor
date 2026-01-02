package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	PollInterval   = 10 * time.Second
	MaxWorkers     = 5
	RequestTimeout = 2 * time.Second
	MaxRetries     = 2
	ErrorThreshold = 0.2
)

type Monitor struct {
	store *ServiceStore
}

func NewMonitor(store *ServiceStore) *Monitor {
	return &Monitor{store: store}
}

func (m *Monitor) Start(ctx context.Context) {
	go func() {
		for {
			start := time.Now()

			services := m.store.List()
			m.runCycle(ctx, services)

			elapsed := time.Since(start)
			if elapsed < PollInterval {
				select {
				case <-ctx.Done():
					return
				case <-time.After(PollInterval - elapsed):
				}
			}
		}
	}()
}

func (m *Monitor) runCycle(ctx context.Context, services []*Service) {
	jobs := make(chan *Service, len(services))
	var wg sync.WaitGroup

	client := &http.Client{Timeout: RequestTimeout}

	// Worker pool
	for i := 0; i < MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for svc := range jobs {
				m.pollService(ctx, client, svc)
			}
		}()
	}

	// Enqueue snapshot
	for _, svc := range services {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- svc:
		}
	}

	close(jobs)
	wg.Wait()
}

func (m *Monitor) pollService(
	ctx context.Context,
	client *http.Client,
	svc *Service,
) {
	now := time.Now()
	metrics := svc.Metrics

	// pessimistic defaults
	metrics.Ready = false
	metrics.LastCheckedAt = now

	// ---- Request with retry ----
	var resp *http.Response
	var err error

	for i := 0; i <= MaxRetries; i++ {
		req, _ := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			svc.URL+"/health",
			nil,
		)
		resp, err = client.Do(req)
		if err == nil {
			break
		}
	}

	// ---- Network failure ----
	if err != nil {
		metrics.Health = Dead
		svc.Metrics = metrics
		m.store.Update(svc)
		return
	}
	defer resp.Body.Close()

	// ---- HTTP but not ready ----
	if resp.StatusCode != http.StatusOK {
		metrics.Health = Unhealthy
		svc.Metrics = metrics
		m.store.Update(svc)
		return
	}

	// ---- Ready ----
	metrics.Ready = true

	var hr HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		metrics.Health = Unhealthy
		svc.Metrics = metrics
		m.store.Update(svc)
		return
	}

	metrics.UptimeSec = hr.UptimeSec
	metrics.RequestCount = hr.RequestCount
	metrics.ErrorCount = hr.ErrorCount

	if hr.RequestCount > 0 {
		metrics.ErrorRate = float64(hr.ErrorCount) / float64(hr.RequestCount)
	}

	if metrics.Attributes == nil {
		metrics.Attributes = map[string]string{}
	}
	for k, v := range hr.Attributes {
		metrics.Attributes[k] = v
	}

	if metrics.ErrorRate > ErrorThreshold {
		metrics.Health = Degraded
	} else {
		metrics.Health = Healthy
	}

	metrics.LastCheckedAge = 0
	svc.Metrics = metrics
	m.store.Update(svc)
}
