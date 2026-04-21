package domain

import "time"

type Task struct {
	ID         uint
	Status     string
	S3Key      string
	Transcript string
	Summary    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
