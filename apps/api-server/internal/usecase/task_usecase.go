package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"speech.local/apps/api-server/internal/repository"
	"speech.local/packages/db/models"
)

const (
	s3UploadPath = "uploads/audio/"
)

type TaskUseCase interface {
	// GetAudioUploadURL 產生音訊上傳的預先簽章網址
	GetAudioUploadURL(ctx context.Context, fileExtension, contentType string) (uploadURL string, s3Key string, err error)
	// ConfirmTask 確認音訊已上傳，並建立轉譯任務
	ConfirmTask(ctx context.Context, s3Key string) (taskID uint, err error)
	// GetTaskDetail 取得任務詳情
	GetTaskDetail(ctx context.Context, id uint) (*models.Task, error)
	// StreamTaskSummary 訂閱任務摘要的串流
	StreamTaskSummary(ctx context.Context, taskID uint) (<-chan string, func() error, error)
}

var (
	ErrInvalidFileExtension = errors.New("invalid file extension")
	ErrInvalidS3Key         = errors.New("invalid s3 key")
)

var supportedExtensions = []string{".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm"}

type taskUseCase struct {
	storageRepo repository.StorageRepo
	taskRepo    repository.TaskRepo
	pubSubRepo  repository.PubSubRepo
}

func NewTaskUseCase(storageRepo repository.StorageRepo, taskRepo repository.TaskRepo, pubSubRepo repository.PubSubRepo) TaskUseCase {
	return &taskUseCase{
		storageRepo: storageRepo,
		taskRepo:    taskRepo,
		pubSubRepo:  pubSubRepo,
	}
}

func (u *taskUseCase) GetAudioUploadURL(ctx context.Context, fileExtension, contentType string) (string, string, error) {
	ext := normalizeExtension(fileExtension)
	if !slices.Contains(supportedExtensions, ext) {
		return "", "", ErrInvalidFileExtension
	}

	filename := uuid.New().String()
	s3Key := fmt.Sprintf("%s%s%s", s3UploadPath, filename, ext)

	uploadURL, err := u.storageRepo.GenerateUploadURL(ctx, s3Key, contentType)
	if err != nil {
		return "", "", err
	}

	return uploadURL, s3Key, nil
}

func normalizeExtension(ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	return "." + strings.ToLower(ext)
}

func (u *taskUseCase) ConfirmTask(ctx context.Context, s3Key string) (uint, error) {
	if s3Key == "" {
		return 0, ErrInvalidS3Key
	}
	return u.taskRepo.CreateTaskWithOutbox(ctx, s3Key)
}

func (u *taskUseCase) GetTaskDetail(ctx context.Context, id uint) (*models.Task, error) {
	return u.taskRepo.GetByID(ctx, id)
}

func (u *taskUseCase) StreamTaskSummary(ctx context.Context, taskID uint) (<-chan string, func() error, error) {
	channel := fmt.Sprintf("task:%d:stream", taskID)
	return u.pubSubRepo.Subscribe(ctx, channel)
}
