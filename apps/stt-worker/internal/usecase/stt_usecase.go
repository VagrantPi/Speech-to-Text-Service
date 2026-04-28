package usecase

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"speech.local/apps/stt-worker/internal/repository"
	"speech.local/packages/config"
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
	env         string
}

func NewSTTUseCase(storageRepo repository.StorageRepo, sttRepo repository.STTRepo, taskRepo repository.TaskRepo, logger *zap.Logger, env string) (STTUseCase, error) {
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
		env:         env,
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

	log.Info("ENV: env mode", zap.String("env", uc.env))

	if uc.env == config.EnvMock {
		localFilePath = "/tmp/mock-audio.wav"
		log.Info("ENV: using mock path")
	} else {
		log.Info("ENV: attempting download", zap.String("s3_key", s3Key))
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
		log.Info("ENV: downloaded file", zap.String("path", localFilePath))

		// Debug: check if file exists
		if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
			log.Error("ENV: file does not exist after download!", zap.String("path", localFilePath))
		} else {
			log.Info("ENV: file exists", zap.String("path", localFilePath))
		}
		// defer os.Remove(localFilePath)
	}

	log.Info("ENV: about to transcribe", zap.String("path", localFilePath))
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
