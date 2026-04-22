package worker

import (
	"context"
	"log"
	"time"

	"speech.local/apps/outbox-relay/internal/usecase"
)

const batchSize = 50

type RelayWorker struct {
	relayService usecase.RelayService
}

func NewRelayWorker(relayService usecase.RelayService) *RelayWorker {
	return &RelayWorker{
		relayService: relayService,
	}
}

func (w *RelayWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	log.Println("relay worker started, polling every 500ms")

	for {
		select {
		case <-ctx.Done():
			log.Println("relay worker shutting down...")
			return
		case <-ticker.C:
			_ = w.relayService.PollAndRelay(ctx)
		}
	}
}
