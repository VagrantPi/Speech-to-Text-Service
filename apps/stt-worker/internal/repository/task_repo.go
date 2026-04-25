package repository

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db"
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
	_, err := db.ExecuteWithOutbox(
		r.db,
		1,
		"llm-queue",
		map[string]interface{}{"task_id": taskID, "transcript": transcript},
		func(tx *gorm.DB) (uint, error) {
			err := tx.WithContext(ctx).Model(&models.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
				"transcript": transcript,
				"status":     "PROCESSING",
			}).Error
			if err != nil {
				return 0, err
			}
			return taskID, nil
		},
	)
	return err
}
