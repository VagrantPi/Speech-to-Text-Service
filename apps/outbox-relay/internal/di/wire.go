//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/google/wire"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"speech.local/apps/outbox-relay/internal/repository"
	"speech.local/apps/outbox-relay/internal/usecase"
	"speech.local/packages/config"
	"speech.local/packages/db"
	"speech.local/packages/mq"
	"speech.local/packages/telemetry"
)

var ProviderSet = wire.NewSet(
	// 1. 基礎設施配置與 DB 連線
	NewAppConfig,
	ProvideDB,

	// 2. MQ 注入與介面綁定 (Duck Typing)
	ProvideRabbitMQPublisher,
	wire.Bind(new(repository.Publisher), new(*mq.RabbitMQPublisher)),

	// 3. DB DAO 注入與介面綁定
	// 使用重構後位於 packages/db 的具體實作
	db.NewOutboxDAO,
	wire.Bind(new(repository.OutboxRepo), new(*db.OutboxDAO)),

	// 4. Usecase 注入
	NewLogger,
	NewRelayService,
)

// NewAppConfig 載入環境變數
func NewAppConfig() (*config.AppConfig, error) {
	return config.NewAppConfig()
}

// ProvideTelemetryConfig 提取 TelemetryConfig
func ProvideTelemetryConfig(cfg *config.AppConfig) telemetry.Config {
	return cfg.TelemetryConfig
}

// ProvideDB 從 AppConfig 提取 DBConfig 並建立 *gorm.DB
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

// ProvideRabbitMQPublisher 從 AppConfig 提取 MQURL 並建立 RabbitMQ 實作
func ProvideRabbitMQPublisher(cfg *config.AppConfig) (*mq.RabbitMQPublisher, error) {
	return mq.NewRabbitMQPublisher(cfg.MQURL)
}

// NewRelayService 建立業務邏輯層
func NewRelayService(publisher repository.Publisher, outboxRepo repository.OutboxRepo, logger *zap.Logger) (usecase.RelayService, error) {
	return usecase.NewRelayService(publisher, outboxRepo, logger)
}

func NewLogger(cfg *config.AppConfig) (*zap.Logger, error) {
	return telemetry.NewLogger(cfg.TelemetryConfig.ServiceName)
}

// InitializeRelayService 初始化整個 Relay 服務的依賴注入
func InitializeRelayService() (usecase.RelayService, error) {
	wire.Build(ProviderSet)
	return nil, nil
}

// InitializeTelemetry 初始化 OpenTelemetry
func InitializeTelemetry(cfg telemetry.Config) (func(context.Context) error, error) {
	return telemetry.InitTracer(cfg)
}
