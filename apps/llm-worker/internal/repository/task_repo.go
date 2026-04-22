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
	_, err := db.ExecuteWithOutbox(
		r.db,
		1,
		"task.completed",
		map[string]interface{}{"task_id": taskID},
		func(tx *gorm.DB) (uint, error) {
			err := tx.WithContext(ctx).Model(&models.Task{}).Where("id = ?", taskID).Updates(map[string]interface{}{
				"summary": summary,
				"status":  "COMPLETED",
			}).Error
			if err != nil {
				return 0, err
			}
			return taskID, nil
		},
	)
	return err
}
