package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Handler exposes HTTP endpoints for registration and metrics
type Handler struct {
	store *ServiceStore
}

func NewHandler(store *ServiceStore) *Handler {
	return &Handler{store: store}
}

// POST /register
// Accepts ServiceRegistration JSON and registers the service.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req ServiceRegistration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ID == "" || req.URL == "" {
		http.Error(w, "id and url are required", http.StatusBadRequest)
		return
	}

	metrics := ServiceMetrics{
		Ready:           false,
		Health:          Unhealthy,
		Attributes:      map[string]string{},
		PollIntervalSec: req.PollIntervalSec,
		LastCheckedAt:   time.Time{},
		LastPolledAt:    time.Time{},
	}
	// copy registration attributes
	for k, v := range req.Attributes {
		metrics.Attributes[k] = v
	}

	svc := &Service{
		ID:      req.ID,
		URL:     req.URL,
		Metrics: metrics,
	}

	h.store.Register(svc)
	w.WriteHeader(http.StatusCreated)
}

// GET /metrics
// Returns all registered service metrics
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	services := h.store.List()
	resp := make([]ServiceMetricsResponse, 0, len(services))

	for _, svc := range services {
		resp = append(resp, toMetricsResponse(svc))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func toMetricsResponse(svc *Service) ServiceMetricsResponse {
	status := "DOWN"
	if svc.Metrics.Ready {
		status = "UP"
	}

	age := int(time.Since(svc.Metrics.LastCheckedAt).Seconds())

	return ServiceMetricsResponse{
		Service:     svc.ID,
		Status:      status,
		Health:      string(svc.Metrics.Health),
		Ready:       svc.Metrics.Ready,
		LastChecked: fmt.Sprintf("%ds ago", age),
		ErrorRate:   svc.Metrics.ErrorRate,
		UptimeSec:   svc.Metrics.UptimeSec,
		Attributes:  svc.Metrics.Attributes,
	}
}
