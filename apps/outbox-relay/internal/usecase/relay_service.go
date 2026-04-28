package usecase

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"speech.local/apps/outbox-relay/internal/repository"
	"speech.local/packages/db/models"
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

	processedCount := 0
	publishFailed := 0
	deleteFailed := 0

	_, err := s.repo.FetchAndProcess(ctx, batchSize, func(txCtx context.Context, tx *gorm.DB, events []*models.OutboxEvent) error {
		for _, event := range events {
			if err := s.publisher.Publish(txCtx, event.Topic, event.Payload); err != nil {
				log.Warn("ProcessBatch: publish failed",
					zap.Uint("event_id", event.ID),
					zap.String("topic", event.Topic),
					zap.Error(err),
				)
				s.failed.Add(txCtx, 1, metric.WithAttributes(attribute.String("status", "publish_failed")))
				publishFailed++
				continue
			}

			if err := s.repo.MarkAsProcessedTx(txCtx, tx, []uint{event.ID}); err != nil {
				log.Warn("ProcessBatch: mark as processed failed",
					zap.Uint("event_id", event.ID),
					zap.Error(err),
				)
				s.failed.Add(txCtx, 1, metric.WithAttributes(attribute.String("status", "delete_failed")))
				deleteFailed++
				continue
			}

			processedCount++
			s.processed.Add(txCtx, 1, metric.WithAttributes(attribute.String("status", "success")))
		}
		return nil
	})

	log.Info("ProcessBatch: finished",
		zap.Int("processed_count", processedCount),
		zap.Int("publish_failed", publishFailed),
		zap.Int("delete_failed", deleteFailed),
	)

	return processedCount, err
}

func (s *relayService) PollAndRelay(ctx context.Context) error {
	log := telemetry.WithTraceID(ctx, s.logger)

	_, err := s.repo.FetchAndProcess(ctx, 50, func(txCtx context.Context, tx *gorm.DB, events []*models.OutboxEvent) error {
		for _, event := range events {
			log.Info("Processing event", zap.Uint("id", event.ID), zap.String("topic", event.Topic))
			if err := s.publisher.Publish(txCtx, event.Topic, event.Payload); err != nil {
				s.failed.Add(txCtx, 1, metric.WithAttributes(attribute.String("status", "publish_failed")))
				return err
			}
			if err := s.repo.MarkAsProcessedTx(txCtx, tx, []uint{event.ID}); err != nil {
				log.Warn("PollAndRelay: failed to mark as processed",
					zap.Uint("event_id", event.ID),
					zap.Error(err),
				)
				return err
			}
			s.processed.Add(txCtx, 1, metric.WithAttributes(attribute.String("status", "success")))
		}
		return nil
	})

	if err != nil {
		log.Error("PollAndRelay: batch failed, rolling back", zap.Error(err))
	}
	return nil
}

var _ RelayService = (*relayService)(nil)
