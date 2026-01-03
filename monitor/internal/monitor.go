package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const (
	// default behavior values
	GlobalPollInterval = 10 * time.Second // pacing for the scheduler loop
	MaxWorkers         = 5
	RequestTimeout     = 2 * time.Second
	MaxRetries         = 2
	ErrorThreshold     = 0.2
)

// Monitor orchestrates polling cycles
type Monitor struct {
	store *ServiceStore
}

func NewMonitor(store *ServiceStore) *Monitor {
	return &Monitor{store: store}
}

// Start the monitor loop in a goroutine. It is self-timed (no overlapping cycles)
// and honors per-service poll intervals.
// ctx cancellation is checked frequently for graceful shutdown.
func (m *Monitor) Start(ctx context.Context) {
	go func() {
		for {
			start := time.Now()

			// gather due services according to per-service PollIntervalSec
			all := m.store.List()
			due := make([]*Service, 0, len(all))
			now := time.Now()
			for _, svc := range all {
				intervalSec := svc.Metrics.PollIntervalSec
				if intervalSec <= 0 {
					intervalSec = int(GlobalPollInterval.Seconds()) // default fallback
				}
				// if never polled, or enough time passed -> due
				if svc.Metrics.LastPolledAt.IsZero() || now.Sub(svc.Metrics.LastPolledAt) >= time.Duration(intervalSec)*time.Second {
					due = append(due, svc)
				}
			}

			// run a single cycle for the due snapshot
			m.runCycle(ctx, due)

			// compute elapsed and decide sleep; always check ctx.Done()
			elapsed := time.Since(start)
			sleep := GlobalPollInterval - elapsed
			if sleep > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(sleep):
				}
			} else {
				// we are behind schedule; check ctx and continue immediately
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}
	}()
}

// runCycle creates a per-cycle job channel sized to number of services and a worker pool
func (m *Monitor) runCycle(ctx context.Context, services []*Service) {
	if len(services) == 0 {
		return
	}
	jobs := make(chan *Service, len(services))

	// start workers
	for i := 0; i < MaxWorkers; i++ {
		go func() {
			client := &http.Client{Timeout: RequestTimeout}
			for {
				select {
				case <-ctx.Done():
					return
				case svc, ok := <-jobs:
					if !ok {
						return
					}
					m.pollService(ctx, client, svc)
				}
			}
		}()
	}

	// enqueue snapshot
	for _, svc := range services {
		select {
		case <-ctx.Done():
			close(jobs)
			return
		case jobs <- svc:
		}
	}

	close(jobs)
	// workers will exit when jobs closed or ctx canceled
}

// pollService performs the actual HTTP call, sets Ready, merges attributes, and computes health.
// It updates LastPolledAt and LastCheckedAt and writes back into the store.
func (m *Monitor) pollService(ctx context.Context, client *http.Client, svc *Service) {
	now := time.Now()
	metrics := svc.Metrics // copy for mutation

	// default pessimistic values
	metrics.Ready = false
	metrics.LastCheckedAt = now
	metrics.LastPolledAt = now

	// perform request with retries
	var resp *http.Response
	var err error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, svc.URL+"/health", nil)
		resp, err = client.Do(req)
		if err == nil {
			break
		}
	}

	if err != nil {
		metrics.Health = Dead
		// update last polled time and persist
		svc.Metrics = metrics
		m.store.Update(svc)
		return
	}
	defer resp.Body.Close()

	// non-200 -> not ready and unhealthy
	if resp.StatusCode != http.StatusOK {
		metrics.Ready = false
		metrics.Health = Unhealthy
		svc.Metrics = metrics
		m.store.Update(svc)
		return
	}

	// 200 OK -> service is Ready
	metrics.Ready = true

	var hr HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		metrics.Health = Unhealthy
		svc.Metrics = metrics
		m.store.Update(svc)
		return
	}

	// fill metrics from response
	metrics.UptimeSec = hr.UptimeSec
	metrics.RequestCount = hr.RequestCount
	metrics.ErrorCount = hr.ErrorCount
	if hr.RequestCount > 0 {
		metrics.ErrorRate = float64(hr.ErrorCount) / float64(hr.RequestCount)
	} else {
		metrics.ErrorRate = 0
	}

	// merge attributes (health response overrides registration values)
	if metrics.Attributes == nil {
		metrics.Attributes = map[string]string{}
	}
	for k, v := range hr.Attributes {
		metrics.Attributes[k] = v
	}

	// determine health
	if metrics.ErrorRate > ErrorThreshold {
		metrics.Health = Degraded
	} else {
		metrics.Health = Healthy
	}

	metrics.LastCheckedAge = 0
	// persist updated metrics (LastPolledAt already set above)
	svc.Metrics = metrics
	m.store.Update(svc)
}
