package internal

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	store *ServiceStore
}

func NewHandler(store *ServiceStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req ServiceRegistration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	svc := &Service{
		ID:  req.ID,
		URL: req.URL,
		Metrics: ServiceMetrics{
			Health:     Unhealthy,
			Ready:      false,
			Attributes: req.Attributes,
		},
	}

	h.store.Register(svc)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(h.store.List())
}
