package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

var start = time.Now()
var req, errc int

func main() {
	go register()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		req++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uptime_sec":    int(time.Since(start).Seconds()),
			"request_count": req,
			"error_count":   errc,
			"attributes": map[string]string{
				"version": "1.0.0",
				"region":  "in",
				"port":    "9001",
			},
		})
	})

	http.ListenAndServe(":9001", nil)
}

func register() {
	time.Sleep(time.Second)
	body, _ := json.Marshal(map[string]interface{}{
		"id":  "svc1",
		"url": "http://localhost:9001",
		"attributes": map[string]string{
			"service": "svc1",
			"env":     "prod",
		},
	})
	http.Post("http://localhost:8080/register", "application/json", bytes.NewBuffer(body))
}
