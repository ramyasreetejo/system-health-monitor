package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var (
	startTime    = time.Now()
	requestCount = 0
	errorCount   = 0
)

type HealthResponse struct {
	UptimeSec    int               `json:"uptime_sec"`
	RequestCount int               `json:"request_count"`
	ErrorCount   int               `json:"error_count"`
	Attributes   map[string]string `json:"attributes"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	requestCount++

	// simulate rare error
	if rand.Float64() < 0.05 {
		errorCount++
		http.Error(w, "temporary error", http.StatusInternalServerError)
		return
	}

	resp := HealthResponse{
		UptimeSec:    int(time.Since(startTime).Seconds()),
		RequestCount: requestCount,
		ErrorCount:   errorCount,
		Attributes: map[string]string{
			"service": "svc1",
			"version": "1.0.0",
			"port":    "9001",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func register() {
	payload := map[string]interface{}{
		"id":                "svc1",
		"url":               "http://localhost:9001",
		"poll_interval_sec": 10,
		"attributes": map[string]string{
			"env":  "dev",
			"team": "core",
		},
	}

	b, _ := json.Marshal(payload)
	_, err := http.Post("http://localhost:8080/register", "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.Fatalf("registration failed: %v", err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	register()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	log.Println("svc1 running on :9001")
	log.Fatal(http.ListenAndServe(":9001", mux))
}
