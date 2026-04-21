package worker

import (
	"context"
	"time"

	"speech.local/apps/outbox-relay/internal/usecase"
)

const batchSize = 100

type RelayWorker struct {
	relayService usecase.RelayService
}

func NewRelayWorker(relayService usecase.RelayService) *RelayWorker {
	return &RelayWorker{
		relayService: relayService,
	}
}

func (w *RelayWorker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			w.processBatch(ctx)
		}
	}
}

func (w *RelayWorker) processBatch(ctx context.Context) {
	processedCount, err := w.relayService.ProcessBatch(ctx, batchSize)
	if err != nil {
		time.Sleep(1 * time.Second)
		return
	}

	if processedCount > 0 {
		return
	}

	time.Sleep(1 * time.Second)
}
