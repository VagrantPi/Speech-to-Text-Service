package models

import (
	"time"
)

// Task 任務主表
type Task struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	Status     string    `gorm:"type:varchar(50);index;not null"`
	S3Key      string    `gorm:"type:varchar(255);not null"`
	Transcript string    `gorm:"type:text"` // 允許 NULL，STT 完成後寫入
	Summary    string    `gorm:"type:text"` // 允許 NULL，LLM 完成後寫入
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}
