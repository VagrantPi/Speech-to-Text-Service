package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"speech.local/apps/llm-worker/internal/repository"
	"speech.local/packages/telemetry"
)

type LLMUseCase interface {
	ProcessTask(ctx context.Context, taskID uint) error
}

type llmUseCase struct {
	taskRepo   repository.TaskRepo
	llmRepo    repository.LLMRepo
	pubsubRepo repository.PubSubRepo
	logger     *zap.Logger
	processed  metric.Int64Counter
	failed     metric.Int64Counter
}

func NewLLMUseCase(taskRepo repository.TaskRepo, llmRepo repository.LLMRepo, pubsubRepo repository.PubSubRepo, logger *zap.Logger) (LLMUseCase, error) {
	processed, err := telemetry.NewCounter("llm_tasks_processed_total", "Total number of LLM tasks processed")
	if err != nil {
		return nil, err
	}
	failed, err := telemetry.NewCounter("llm_tasks_failed_total", "Total number of LLM tasks failed")
	if err != nil {
		return nil, err
	}

	return &llmUseCase{
		taskRepo:   taskRepo,
		llmRepo:    llmRepo,
		pubsubRepo: pubsubRepo,
		logger:     logger,
		processed:  processed,
		failed:     failed,
	}, nil
}

func (u *llmUseCase) ProcessTask(ctx context.Context, taskID uint) error {
	log := telemetry.WithTraceID(ctx, u.logger)
	log.Info("ProcessTask: starting",
		zap.Uint("task_id", taskID),
	)

	transcript, err := u.taskRepo.GetTranscript(ctx, taskID)
	if err != nil {
		log.Error("ProcessTask: failed to get transcript",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		u.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "get_transcript_failed")))
		return fmt.Errorf("failed to get transcript: %w", err)
	}

	tokenChan := make(chan string)

	go func() {
		for token := range tokenChan {
			payload := map[string]string{"token": token}
			jsonStr, _ := json.Marshal(payload)
			_ = u.pubsubRepo.Publish(ctx, fmt.Sprintf("task:%d:stream", taskID), string(jsonStr))
		}
	}()

	fullSummary, err := u.llmRepo.GenerateSummaryStream(ctx, transcript, tokenChan)
	if err != nil {
		log.Error("ProcessTask: failed to generate summary",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		u.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "generate_summary_failed")))
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	donePayload, _ := json.Marshal(map[string]string{"token": "[DONE]"})
	_ = u.pubsubRepo.Publish(ctx, fmt.Sprintf("task:%d:stream", taskID), string(donePayload))

	if err := u.taskRepo.UpdateSummary(ctx, taskID, fullSummary); err != nil {
		log.Error("ProcessTask: failed to update summary",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		u.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "update_summary_failed")))
		return fmt.Errorf("failed to update summary: %w", err)
	}

	log.Info("ProcessTask: completed",
		zap.Uint("task_id", taskID),
	)
	u.processed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))

	return nil
}
