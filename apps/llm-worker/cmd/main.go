package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"speech.local/apps/llm-worker/internal/di"
	"speech.local/packages/config"
	"speech.local/packages/telemetry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	worker, err := di.InitializeWorker()
	if err != nil {
		log.Fatalf("Failed to initialize worker: %v", err)
	}

	log.Println("LLM Worker starting...")

	if err := worker.Start(ctx); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
