package internal

import "time"

type HealthStatus string

const (
	Healthy   HealthStatus = "healthy"
	Degraded  HealthStatus = "degraded"
	Unhealthy HealthStatus = "unhealthy"
	Dead      HealthStatus = "dead"
)

type ServiceRegistration struct {
	ID         string            `json:"id"`
	URL        string            `json:"url"`
	Attributes map[string]string `json:"attributes"`
}

type HealthResponse struct {
	UptimeSec    int               `json:"uptime_sec"`
	RequestCount int               `json:"request_count"`
	ErrorCount   int               `json:"error_count"`
	Attributes   map[string]string `json:"attributes"`
}

type ServiceMetrics struct {
	Ready          bool              `json:"ready"`
	Health         HealthStatus      `json:"health"`
	UptimeSec      int               `json:"uptime_sec"`
	RequestCount   int               `json:"request_count"`
	ErrorCount     int               `json:"error_count"`
	ErrorRate      float64           `json:"error_rate"`
	LastCheckedAge int               `json:"last_checked_age_sec"`
	Attributes     map[string]string `json:"attributes"`
	LastCheckedAt  time.Time         `json:"-"`
}

type Service struct {
	ID      string         `json:"id"`
	URL     string         `json:"url"`
	Metrics ServiceMetrics `json:"metrics"`
}
