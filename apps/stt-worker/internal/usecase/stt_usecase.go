package usecase

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"speech.local/apps/stt-worker/internal/repository"
	"speech.local/packages/telemetry"
)

type STTUseCase interface {
	ProcessTask(ctx context.Context, taskID uint, s3Key string) error
}

type sttUseCase struct {
	storageRepo repository.StorageRepo
	sttRepo     repository.STTRepo
	taskRepo    repository.TaskRepo
	logger      *zap.Logger
	processed   metric.Int64Counter
	failed      metric.Int64Counter
	debug       bool
}

func NewSTTUseCase(storageRepo repository.StorageRepo, sttRepo repository.STTRepo, taskRepo repository.TaskRepo, logger *zap.Logger, debug bool) (STTUseCase, error) {
	processed, err := telemetry.NewCounter("stt_tasks_processed_total", "Total number of STT tasks processed")
	if err != nil {
		return nil, err
	}
	failed, err := telemetry.NewCounter("stt_tasks_failed_total", "Total number of STT tasks failed")
	if err != nil {
		return nil, err
	}

	return &sttUseCase{
		storageRepo: storageRepo,
		sttRepo:     sttRepo,
		taskRepo:    taskRepo,
		logger:      logger,
		processed:   processed,
		failed:      failed,
		debug:       debug,
	}, nil
}

func (uc *sttUseCase) ProcessTask(ctx context.Context, taskID uint, s3Key string) error {
	log := telemetry.WithTraceID(ctx, uc.logger)
	log.Info("ProcessTask: starting",
		zap.Uint("task_id", taskID),
		zap.String("s3_key", s3Key),
	)

	var localFilePath string
	var err error

	if uc.debug {
		localFilePath = "/tmp/mock-audio.wav"
	} else {
		localFilePath, err = uc.storageRepo.DownloadToTempFile(ctx, s3Key)
		if err != nil {
			log.Error("ProcessTask: failed to download",
				zap.Uint("task_id", taskID),
				zap.String("s3_key", s3Key),
				zap.Error(err),
			)
			uc.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "download_failed")))
			return err
		}
		defer os.Remove(localFilePath)
	}

	transcript, err := uc.sttRepo.Transcribe(ctx, localFilePath)
	if err != nil {
		log.Error("ProcessTask: failed to transcribe",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		uc.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "transcribe_failed")))
		return err
	}

	if err := uc.taskRepo.UpdateTranscript(ctx, taskID, transcript); err != nil {
		log.Error("ProcessTask: failed to update transcript",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		uc.failed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "update_transcript_failed")))
		return err
	}

	log.Info("ProcessTask: completed",
		zap.Uint("task_id", taskID),
	)
	uc.processed.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))

	return nil
}
