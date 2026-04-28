package worker

import (
	"context"
	"encoding/json"
	"log"

	"speech.local/apps/stt-worker/internal/usecase"
)

type STTWorker struct {
	consumer MQConsumer
	uc       usecase.STTUseCase
}

type taskPayload struct {
	TaskID uint   `json:"task_id"`
	S3Key  string `json:"s3_key"`
}

func NewSTTWorker(consumer MQConsumer, uc usecase.STTUseCase) *STTWorker {
	return &STTWorker{
		consumer: consumer,
		uc:       uc,
	}
}

func (w *STTWorker) Start(ctx context.Context) error {
	log.Println("Starting STT Worker...")

	return w.consumer.Consume(ctx, "stt-queue", w.handleMessage)
}

func (w *STTWorker) handleMessage(ctx context.Context, payload []byte) error {
	log.Printf("[STT Worker] Received message: %s", string(payload))

	var task taskPayload
	if err := json.Unmarshal(payload, &task); err != nil {
		log.Printf("Failed to parse task payload: %v", err)
		return err
	}

	log.Printf("Processing task ID=%d, s3_key=%s", task.TaskID, task.S3Key)

	return w.uc.ProcessTask(ctx, task.TaskID, task.S3Key)
}
