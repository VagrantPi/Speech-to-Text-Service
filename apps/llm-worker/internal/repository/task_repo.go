package repository

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db"
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
	taskDAO := db.NewTaskDAO(r.db)
	task, err := taskDAO.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	task.Summary = summary
	task.Status = "COMPLETED"
	return taskDAO.Update(ctx, task)
}
