package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"speech.local/apps/stt-worker/internal/di"
	"speech.local/packages/config"
	"speech.local/packages/telemetry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down worker...")
		cancel()
	}()

	telemetryCfg := telemetry.Config{}
	if err := config.LoadConfig(&telemetryCfg); err != nil {
		log.Fatalf("failed to load telemetry config: %v", err)
	}

	shutdown, err := telemetry.Init(telemetryCfg)
	if err != nil {
		log.Fatalf("failed to initialize telemetry: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("failed to shutdown tracer: %v", err)
		}
	}()

	w, err := di.InitializeWorker()
	if err != nil {
		log.Fatalf("Failed to initialize worker: %v", err)
	}

	log.Println("STT Worker initialized")

	if err := w.Start(ctx); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}
