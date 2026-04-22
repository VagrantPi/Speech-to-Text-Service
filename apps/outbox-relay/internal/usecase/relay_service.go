package usecase

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"speech.local/apps/outbox-relay/internal/repository"
	"speech.local/packages/telemetry"
)

type RelayService interface {
	ProcessBatch(ctx context.Context, batchSize int) (int, error)
	PollAndRelay(ctx context.Context) error
}

type relayService struct {
	publisher repository.Publisher
	repo      repository.OutboxRepo
	logger    *zap.Logger
	processed metric.Int64Counter
	failed    metric.Int64Counter
	duration  metric.Float64Histogram
}

func NewRelayService(publisher repository.Publisher, repo repository.OutboxRepo, logger *zap.Logger) (RelayService, error) {
	processed, err := telemetry.NewCounter("outbox_events_processed_total", "Total number of outbox events processed")
	if err != nil {
		return nil, err
	}
	failed, err := telemetry.NewCounter("outbox_events_failed_total", "Total number of outbox events failed")
	if err != nil {
		return nil, err
	}
	duration, err := telemetry.NewHistogram("outbox_batch_duration_seconds", "Outbox batch processing duration in seconds", []float64{0.01, 0.05, 0.1, 0.5, 1.0})
	if err != nil {
		return nil, err
	}

	return &relayService{
		publisher: publisher,
		repo:      repo,
		logger:    logger,
		processed: processed,
		failed:    failed,
		duration:  duration,
	}, nil
}

func (s *relayService) ProcessBatch(ctx context.Context, batchSize int) (int, error) {
	log := telemetry.WithTraceID(ctx, s.logger)
	log.Info("ProcessBatch: starting",
		zap.Int("batch_size", batchSize),
	)

	events, err := s.repo.FetchPendingEvents(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	processedCount := 0
	publishFailed := 0
	deleteFailed := 0

	for _, event := range events {
		if err := s.publisher.Publish(ctx, event.Topic, event.Payload); err != nil {
			log.Warn("ProcessBatch: publish failed",
				zap.Uint("event_id", event.ID),
				zap.String("topic", event.Topic),
				zap.Error(err),
			)
			s.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "publish_failed")))
			publishFailed++
			continue
		}

		if err := s.repo.MarkAsProcessed(ctx, event.ID); err != nil {
			log.Warn("ProcessBatch: delete failed",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
			s.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "delete_failed")))
			deleteFailed++
			continue
		}

		processedCount++
		s.processed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	}

	log.Info("ProcessBatch: finished",
		zap.Int("processed_count", processedCount),
		zap.Int("publish_failed", publishFailed),
		zap.Int("delete_failed", deleteFailed),
	)

	return processedCount, nil
}

func (s *relayService) PollAndRelay(ctx context.Context) error {
	log := telemetry.WithTraceID(ctx, s.logger)

	events, err := s.repo.FetchPendingEvents(ctx, 50)
	if err != nil {
		return err
	}

	for _, event := range events {
		err := s.publisher.Publish(ctx, event.Topic, event.Payload)
		if err != nil {
			s.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "publish_failed")))
			s.repo.MarkAsFailed(ctx, event.ID, err.Error())
			continue
		}

		err = s.repo.MarkAsProcessed(ctx, event.ID)
		if err != nil {
			log.Error("PollAndRelay: failed to mark as processed",
				zap.Uint("event_id", event.ID),
				zap.Error(err),
			)
			return err
		}
		s.processed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	}
	return nil
}

var _ RelayService = (*relayService)(nil)
