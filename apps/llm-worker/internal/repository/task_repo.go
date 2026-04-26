package repository

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

type TaskRepo interface {
	GetTranscript(ctx context.Context, taskID uint) (string, error)
	UpdateSummary(ctx context.Context, taskID uint, summary string) error
}

type taskRepoImpl struct {
	db *gorm.DB
}

func NewTaskRepo(dbConn *gorm.DB) TaskRepo {
	return &taskRepoImpl{db: dbConn}
}

func (r *taskRepoImpl) GetTranscript(ctx context.Context, taskID uint) (string, error) {
	taskDAO := db.NewTaskDAO(r.db)
	task, err := taskDAO.FindByID(ctx, taskID)
	if err != nil {
		return "", err
	}
	return task.Transcript, nil
}

func (r *taskRepoImpl) UpdateSummary(ctx context.Context, taskID uint, summary string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Task{}).Where("id = ? AND status = ?", taskID, "PROCESSING").Updates(map[string]interface{}{
			"summary": summary,
			"status":  "COMPLETED",
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return nil
	})
}
