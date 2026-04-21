//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"speech.local/apps/api-server/internal/handler"
	"speech.local/apps/api-server/internal/repository"
	"speech.local/apps/api-server/internal/usecase"

	"speech.local/packages/config"
	"speech.local/packages/db"
	"speech.local/packages/storage"
)

var ProviderSet = wire.NewSet(
	// 1. 基礎設施配置
	NewAppConfig,
	ProvideDB,

	// 2. Storage 注入與介面綁定 (Duck Typing)
	NewS3Storage,
	wire.Bind(new(repository.StorageRepo), new(*storage.S3Storage)),

	// 3. DB Repository 注入
	// 因為 repository.NewTaskRepo 已經設計成吃 *gorm.DB 並直接回傳 TaskRepo 介面
	// 所以不需要 wire.Bind，直接放進來即可
	repository.NewTaskRepo,

	// 4. Usecase & Handler 注入
	NewTaskUseCase,
	NewTaskHandler,
)

func NewAppConfig() (*config.AppConfig, error) {
	return config.NewAppConfig()
}

// ProvideDB 負責從 Config 提取 DBConfig 並建立 *gorm.DB 連線
func ProvideDB(cfg *config.AppConfig) (*gorm.DB, error) {
	return db.NewPostgresConn(cfg.DBConfig)
}

// NewS3Storage 負責從 Config 提取 S3Config 並建立 S3 Client
func NewS3Storage(cfg *config.AppConfig) (*storage.S3Storage, error) {
	return storage.NewS3Storage(cfg.S3Config)
}

func NewTaskUseCase(storageRepo repository.StorageRepo, taskRepo repository.TaskRepo) usecase.TaskUseCase {
	return usecase.NewTaskUseCase(storageRepo, taskRepo)
}

func NewTaskHandler(taskUsecase usecase.TaskUseCase) *handler.TaskHandler {
	return handler.NewTaskHandler(taskUsecase)
}

// InitializeTaskDependencies injects all dependencies for task handler
func InitializeTaskDependencies() (*handler.TaskHandler, error) {
	wire.Build(ProviderSet)
	return nil, nil
}
