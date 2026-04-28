package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
	"speech.local/packages/db"
	"speech.local/packages/redis"
	"speech.local/packages/storage"
	"speech.local/packages/telemetry"
)

const (
	EnvMock       = "mock"
	EnvLocal      = "local"
	EnvProduction = "production"
)

func LoadConfig(configStruct interface{}) error {
	viper.AddConfigPath(".")
	viper.AddConfigPath("../..")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.SetDefault("ENV", EnvLocal)

	addEnvToViper("DB_HOST", "DB_HOST")
	addEnvToViper("DB_PORT", "DB_PORT")
	addEnvToViper("DB_USER", "DB_USER")
	addEnvToViper("DB_PASSWORD", "DB_PASSWORD")
	addEnvToViper("DB_NAME", "DB_NAME")
	addEnvToViper("DB_SSLMODE", "DB_SSLMODE")
	addEnvToViper("REDIS_HOST", "REDIS_HOST")
	addEnvToViper("REDIS_PASSWORD", "REDIS_PASSWORD")
	addEnvToViper("MQ_URL", "MQ_URL")
	addEnvToViper("MQ_PREFETCH_COUNT", "MQ_PREFETCH_COUNT")
	addEnvToViper("AWS_REGION", "AWS_REGION")
	addEnvToViper("AWS_S3_BUCKET", "AWS_S3_BUCKET")
	addEnvToViper("AWS_ACCESS_KEY", "AWS_ACCESS_KEY")
	addEnvToViper("AWS_SECRET_KEY", "AWS_SECRET_KEY")
	addEnvToViper("AWS_ENDPOINT", "AWS_ENDPOINT")
	addEnvToViper("AWS_PUBLIC_ENDPOINT", "AWS_PUBLIC_ENDPOINT")
	addEnvToViper("EXPIRATION_IN_MINUTES", "EXPIRATION_IN_MINUTES")
	addEnvToViper("OPENAI_API_KEY", "OPENAI_API_KEY")
	addEnvToViper("ENV", "ENV")
	addEnvToViper("API_PORT", "API_PORT")
	addEnvToViper("OTEL_SERVICE_NAME", "OTEL_SERVICE_NAME")
	addEnvToViper("OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("警告: 找不到 .env 檔案，將完全依賴系統環境變數")
		} else {
			return err
		}
	}

	err := viper.Unmarshal(configStruct)
	return err
}

func addEnvToViper(envKey, viperKey string) {
	if val := os.Getenv(envKey); val != "" {
		viper.Set(viperKey, val)
	}
}

// AppConfig 是 API Server 專屬的設定總表
// 使用 mapstructure tag 來對應 Viper 讀取出來的 key
type AppConfig struct {
	APIPort string `mapstructure:"API_PORT"`
	MQURL   string `mapstructure:"MQ_URL"`

	PrefetchCount int `mapstructure:"MQ_PREFETCH_COUNT"`

	AWSRegion           string `mapstructure:"AWS_REGION"`
	AWSS3Bucket         string `mapstructure:"AWS_S3_BUCKET"`
	AWSAccessKey        string `mapstructure:"AWS_ACCESS_KEY"`
	AWSSecretKey        string `mapstructure:"AWS_SECRET_KEY"`
	AWSEndpoint         string `mapstructure:"AWS_ENDPOINT"`
	AWSPublicEndpoint   string `mapstructure:"AWS_PUBLIC_ENDPOINT"`
	ExpirationInMinutes int    `mapstructure:"EXPIRATION_IN_MINUTES"`

	OpenAIAPIKey string `mapstructure:"OPENAI_API_KEY"`
	Env          string `mapstructure:"ENV"`

	S3Config        storage.S3Config `mapstructure:",squash"`
	DBConfig        db.Config        `mapstructure:",squash"`
	RedisConfig     redis.Config     `mapstructure:",squash"`
	TelemetryConfig telemetry.Config `mapstructure:",squash"`
}

func NewAppConfig() (*AppConfig, error) {
	var cfg AppConfig

	err := LoadConfig(&cfg)
	if err != nil {
		return nil, err
	}
	if cfg.PrefetchCount == 0 {
		cfg.PrefetchCount = 1
	}
	return &cfg, nil
}
