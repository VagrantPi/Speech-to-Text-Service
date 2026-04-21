package usecase

import (
	"context"
	"log"

	"speech.local/apps/outbox-relay/internal/repository"
)

type RelayService interface {
	ProcessBatch(ctx context.Context, batchSize int) (int, error)
}

type relayService struct {
	publisher repository.Publisher
	repo      repository.OutboxRepo
}

func NewRelayService(publisher repository.Publisher, repo repository.OutboxRepo) RelayService {
	return &relayService{
		publisher: publisher,
		repo:      repo,
	}
}

func (s *relayService) ProcessBatch(ctx context.Context, batchSize int) (int, error) {
	events, err := s.repo.FetchPendingEvents(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	processedCount := 0
	for _, event := range events {
		if err := s.publisher.Publish(ctx, event.Topic, event.Payload); err != nil {
			log.Printf("failed to publish event %d: %v", event.ID, err)
			continue
		}

		if err := s.repo.Delete(ctx, event.ID); err != nil {
			log.Printf("failed to delete event %d: %v", event.ID, err)
			continue
		}

		processedCount++
	}

	return processedCount, nil
}

var _ RelayService = (*relayService)(nil)
