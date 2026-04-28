//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/google/wire"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"speech.local/apps/stt-worker/internal/repository"
	"speech.local/apps/stt-worker/internal/usecase"
	"speech.local/apps/stt-worker/internal/worker"

	"speech.local/packages/config"
	"speech.local/packages/db"
	"speech.local/packages/mq"
	"speech.local/packages/storage"
	"speech.local/packages/stt"
	"speech.local/packages/telemetry"
)

var ProviderSet = wire.NewSet(
	NewAppConfig,
	ProvideDB,

	NewS3Storage,
	wire.Bind(new(repository.StorageRepo), new(*storage.S3Storage)),

	NewSTTRepo,
	wire.Bind(new(repository.STTRepo), new(stt.STTRepoInterface)),

	NewTaskRepo,

	NewLogger,
	NewSTTUseCase,

	ProvideRabbitMQConsumer,
	wire.Bind(new(worker.MQConsumer), new(*mq.RabbitMQConsumer)),

	worker.NewSTTWorker,
)

func NewAppConfig() (*config.AppConfig, error) {
	return config.NewAppConfig()
}

func ProvideTelemetryConfig(cfg *config.AppConfig) telemetry.Config {
	return cfg.TelemetryConfig
}

func ProvideDB(cfg *config.AppConfig) (*gorm.DB, error) {
	gdb, err := db.NewPostgresConn(cfg.DBConfig)
	if err != nil {
		return nil, err
	}
	if err := gdb.Use(otelgorm.NewPlugin()); err != nil {
		return nil, err
	}
	return gdb, nil
}

func NewS3Storage(cfg *config.AppConfig) (*storage.S3Storage, error) {
	return storage.NewS3Storage(cfg.S3Config)
}

func NewSTTRepo(cfg *config.AppConfig) (stt.STTRepoInterface, error) {
	switch cfg.Env {
	case config.EnvMock:
		return stt.NewMockSTTService(), nil
	case config.EnvLocal:
		return stt.NewLocalSTTService(""), nil
	default:
		return stt.NewOpenAISTTService(cfg.OpenAIAPIKey)
	}
}

func NewTaskRepo(db *gorm.DB) repository.TaskRepo {
	return repository.NewTaskRepo(db)
}

func NewSTTUseCase(storageRepo repository.StorageRepo, sttRepo repository.STTRepo, taskRepo repository.TaskRepo, logger *zap.Logger, cfg *config.AppConfig) (usecase.STTUseCase, error) {
	return usecase.NewSTTUseCase(storageRepo, sttRepo, taskRepo, logger, cfg.Env)
}

func NewLogger(cfg *config.AppConfig) (*zap.Logger, error) {
	return telemetry.NewLogger(cfg.TelemetryConfig.ServiceName)
}

func ProvideRabbitMQConsumer(cfg *config.AppConfig) (*mq.RabbitMQConsumer, error) {
	return mq.NewRabbitMQConsumerWithAMQPAndPrefetch(cfg.MQURL, cfg.PrefetchCount, &mq.RealAMQP{})
}

func InitializeWorker() (*worker.STTWorker, error) {
	wire.Build(ProviderSet)
	return nil, nil
}

func InitializeTelemetry(cfg telemetry.Config) (func(context.Context) error, error) {
	return telemetry.InitTracer(cfg)
}
