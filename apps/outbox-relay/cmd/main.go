package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"speech.local/apps/outbox-relay/internal/di"
	"speech.local/apps/outbox-relay/internal/worker"
	"speech.local/packages/config"
	"speech.local/packages/telemetry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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

	relayService, err := di.InitializeRelayService()
	if err != nil {
		log.Fatalf("failed to initialize dependencies: %v", err)
	}

	relayWorker := worker.NewRelayWorker(relayService)

	log.Println("starting outbox-relay worker...")
	relayWorker.Start(ctx)

	log.Println("outbox-relay worker stopped")
}
