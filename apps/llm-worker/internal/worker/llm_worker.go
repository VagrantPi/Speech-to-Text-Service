package worker

import (
	"context"
	"encoding/json"
	"log"

	"speech.local/apps/llm-worker/internal/usecase"
)

type MQConsumer interface {
	Consume(ctx context.Context, queue string, handler func(ctx context.Context, payload []byte) error) error
}

type LLMWorker struct {
	consumer MQConsumer
	uc       usecase.LLMUseCase
}

type taskPayload struct {
	TaskID uint `json:"task_id"`
}

func NewLLMWorker(consumer MQConsumer, uc usecase.LLMUseCase) *LLMWorker {
	return &LLMWorker{
		consumer: consumer,
		uc:       uc,
	}
}

func (w *LLMWorker) Start(ctx context.Context) error {
	log.Println("Starting LLM Worker...")

	return w.consumer.Consume(ctx, "llm-queue", w.handleMessage)
}

func (w *LLMWorker) handleMessage(ctx context.Context, payload []byte) error {
	var task taskPayload
	if err := json.Unmarshal(payload, &task); err != nil {
		log.Printf("Failed to parse task payload: %v", err)
		return err
	}

	log.Printf("Processing task ID=%d", task.TaskID)

	return w.uc.ProcessTask(ctx, task.TaskID)
}
