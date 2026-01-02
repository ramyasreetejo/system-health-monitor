package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"system-health-monitor/monitor/internal"
)

func main() {
	store := internal.NewServiceStore()
	handler := internal.NewHandler(store)
	monitor := internal.NewMonitor(store)

	ctx, cancel := context.WithCancel(context.Background())
	monitor.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/register", handler.Register)
	mux.HandleFunc("/metrics", handler.Metrics)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Monitor running on :8080")
		server.ListenAndServe()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	cancel()
	ctxShutdown, _ := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(ctxShutdown)
}
