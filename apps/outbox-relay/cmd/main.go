package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"speech.local/apps/outbox-relay/internal/di"
	"speech.local/apps/outbox-relay/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	relayService, err := di.InitializeRelayService()
	if err != nil {
		log.Fatalf("failed to initialize dependencies: %v", err)
	}

	relayWorker := worker.NewRelayWorker(relayService)

	log.Println("starting outbox-relay worker...")
	relayWorker.Start(ctx)

	log.Println("outbox-relay worker stopped")
}
