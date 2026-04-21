package service

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

type TaskService interface {
	CreateTaskWithOutbox(ctx context.Context, s3Key string) (taskID uint, err error)
}

type taskService struct {
	db        *gorm.DB
	taskDAO   *db.TaskDAO
	outboxDAO *db.OutboxDAO
}

func NewTaskService(database *gorm.DB, taskDAO *db.TaskDAO, outboxDAO *db.OutboxDAO) TaskService {
	return &taskService{
		db:        database,
		taskDAO:   taskDAO,
		outboxDAO: outboxDAO,
	}
}

func (r *taskService) CreateTaskWithOutbox(ctx context.Context, s3Key string) (uint, error) {
	var taskID uint
	err := r.db.Transaction(func(tx *gorm.DB) error {
		task := &models.Task{
			S3Key: s3Key,
		}
		if err := r.taskDAO.Create(tx, task); err != nil {
			return err
		}

		taskID = task.ID

		outboxEvent := &models.OutboxEvent{
			AggregateTypeID: models.AggregateTypeTask,
			AggregateID:     task.ID,
			Topic:           "task.created",
			Payload:         []byte{},
			Status:          "PENDING",
		}
		if err := tx.Create(outboxEvent).Error; err != nil {
			return err
		}

		return nil
	})

	return taskID, err
}
