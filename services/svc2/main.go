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
		if req%4 == 0 {
			errc++
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uptime_sec":    int(time.Since(start).Seconds()),
			"request_count": req,
			"error_count":   errc,
			"attributes": map[string]string{
				"version": "2.1.0",
				"region":  "eu",
				"port":    "9002",
			},
		})
	})

	http.ListenAndServe(":9002", nil)
}

func register() {
	time.Sleep(time.Second)
	body, _ := json.Marshal(map[string]interface{}{
		"id":  "svc2",
		"url": "http://localhost:9002",
		"attributes": map[string]string{
			"service": "svc2",
			"env":     "prod",
		},
	})
	http.Post("http://localhost:8080/register", "application/json", bytes.NewBuffer(body))
}
