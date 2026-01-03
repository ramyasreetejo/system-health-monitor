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

	// start http server
	go func() {
		log.Println("monitor running on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down monitor...")
	cancel()

	shutdownCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()
	server.Shutdown(shutdownCtx)
}
