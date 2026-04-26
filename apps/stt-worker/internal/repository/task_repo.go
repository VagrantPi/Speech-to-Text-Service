package repository

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
	"speech.local/packages/db/models"
)

type TaskRepo interface {
	UpdateTranscript(ctx context.Context, taskID uint, transcript string) error
}

type taskRepoImpl struct {
	db *gorm.DB
}

func NewTaskRepo(dbConn *gorm.DB) TaskRepo {
	return &taskRepoImpl{db: dbConn}
}

func (r *taskRepoImpl) UpdateTranscript(ctx context.Context, taskID uint, transcript string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Task{}).Where("id = ? AND status = ?", taskID, "CREATED").Updates(map[string]interface{}{
			"transcript": transcript,
			"status":     "PROCESSING",
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		payload := map[string]interface{}{
			"task_id":    taskID,
			"transcript": transcript,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		outboxEvent := &models.OutboxEvent{
			AggregateTypeID: 1,
			AggregateID:     taskID,
			Topic:           "llm-queue",
			Payload:         payloadBytes,
			Status:          "PENDING",
		}
		return tx.Create(outboxEvent).Error
	})
	return err
}
