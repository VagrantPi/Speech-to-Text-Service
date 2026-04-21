package repository

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db"
	"speech.local/packages/db/models"
)

// TaskRepo 是 api-server 專屬的微型介面 (ISP)
type TaskRepo interface {
	CreateTaskWithOutbox(ctx context.Context, s3Key string) (uint, error)
}

// 這是 api-server 端的具體實作，負責「協調」底層機制與業務領域
type taskRepoImpl struct {
	db *gorm.DB
}

func NewTaskRepo(dbConn *gorm.DB) TaskRepo {
	return &taskRepoImpl{db: dbConn}
}

func (r *taskRepoImpl) CreateTaskWithOutbox(ctx context.Context, s3Key string) (uint, error) {
	// 準備要傳給 outbox 的 payload
	payload := map[string]interface{}{
		"s3_key": s3Key,
		"action": "task_created",
	}

	// 呼叫底層的高階函數，把真正的 CRUD 動作透過閉包傳進去
	return db.ExecuteWithOutbox(r.db, models.AggregateTypeTask, "task.created", payload, func(tx *gorm.DB) (uint, error) {
		task := &models.Task{
			Status: "CREATED",
			S3Key:  s3Key,
		}
		// 這裡才是真正寫入 Task 的地方
		if err := tx.WithContext(ctx).Create(task).Error; err != nil {
			return 0, err
		}
		return task.ID, nil
	})
}
