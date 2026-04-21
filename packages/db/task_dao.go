package db

import (
	"context"

	"gorm.io/gorm"
	"speech.local/packages/db/models"
)

type TaskDAO struct {
	db *gorm.DB
}

func NewTaskDAO(db *gorm.DB) *TaskDAO {
	return &TaskDAO{db: db}
}

func (d *TaskDAO) Create(tx *gorm.DB, task *models.Task) error {
	return tx.Create(task).Error
}

func (d *TaskDAO) FindByID(ctx context.Context, id uint) (*models.Task, error) {
	var task models.Task
	err := d.db.WithContext(ctx).First(&task, id).Error
	return &task, err
}

func (d *TaskDAO) Update(ctx context.Context, task *models.Task) error {
	return d.db.WithContext(ctx).Save(task).Error
}
