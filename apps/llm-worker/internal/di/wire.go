//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/google/wire"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"speech.local/apps/llm-worker/internal/repository"
	"speech.local/apps/llm-worker/internal/usecase"
	"speech.local/apps/llm-worker/internal/worker"

	"speech.local/packages/config"
	"speech.local/packages/db"
	"speech.local/packages/llm"
	"speech.local/packages/mq"
	"speech.local/packages/redis"
	"speech.local/packages/telemetry"
)

var ProviderSet = wire.NewSet(
	NewAppConfig,
	ProvideDB,

	NewLLMRepo,
	wire.Bind(new(repository.LLMRepo), new(llm.LLMRepoInterface)),

	ProvideRedisClient,
	wire.Bind(new(repository.PubSubRepo), new(*redis.RedisClient)),

	NewTaskRepo,

	ProvideRabbitMQConsumer,
	wire.Bind(new(worker.MQConsumer), new(*mq.RabbitMQConsumer)),

	NewLogger,
	NewLLMUseCase,

	worker.NewLLMWorker,
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

func NewLLMRepo(cfg *config.AppConfig) (llm.LLMRepoInterface, error) {
	if cfg.Debug {
		return llm.NewMockLLMService(), nil
	}
	return llm.NewOpenAIStreamService(cfg.OpenAIAPIKey), nil
}

func ProvideRedisClient(cfg *config.AppConfig) (*redis.RedisClient, error) {
	return redis.NewRedisClient(cfg.RedisConfig)
}

func NewTaskRepo(db *gorm.DB) repository.TaskRepo {
	return repository.NewTaskRepo(db)
}

func NewLLMUseCase(taskRepo repository.TaskRepo, llmRepo repository.LLMRepo, pubsubRepo repository.PubSubRepo, logger *zap.Logger) (usecase.LLMUseCase, error) {
	return usecase.NewLLMUseCase(taskRepo, llmRepo, pubsubRepo, logger)
}

func NewLogger(cfg *config.AppConfig) (*zap.Logger, error) {
	return telemetry.NewLogger(cfg.TelemetryConfig.ServiceName)
}

func ProvideRabbitMQConsumer(cfg *config.AppConfig) (*mq.RabbitMQConsumer, error) {
	return mq.NewRabbitMQConsumer(cfg.MQURL)
}

func InitializeWorker() (*worker.LLMWorker, error) {
	wire.Build(ProviderSet)
	return nil, nil
}

func InitializeTelemetry(cfg telemetry.Config) (func(context.Context) error, error) {
	return telemetry.InitTracer(cfg)
}
