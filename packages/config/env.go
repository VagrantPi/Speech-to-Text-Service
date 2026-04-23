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

func LoadConfig(configStruct interface{}) error {
	viper.AddConfigPath(".")
	viper.AddConfigPath("../..")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	addEnvToViper("DB_HOST", "DB_HOST")
	addEnvToViper("DB_PORT", "DB_PORT")
	addEnvToViper("DB_USER", "DB_USER")
	addEnvToViper("DB_PASSWORD", "DB_PASSWORD")
	addEnvToViper("DB_NAME", "DB_NAME")
	addEnvToViper("DB_SSLMODE", "DB_SSLMODE")
	addEnvToViper("REDIS_HOST", "REDIS_HOST")
	addEnvToViper("REDIS_PASSWORD", "REDIS_PASSWORD")
	addEnvToViper("MQ_URL", "MQ_URL")
	addEnvToViper("AWS_REGION", "AWS_REGION")
	addEnvToViper("AWS_S3_BUCKET", "AWS_S3_BUCKET")
	addEnvToViper("AWS_ACCESS_KEY", "AWS_ACCESS_KEY")
	addEnvToViper("AWS_SECRET_KEY", "AWS_SECRET_KEY")
	addEnvToViper("AWS_ENDPOINT", "AWS_ENDPOINT")
	addEnvToViper("EXPIRATION_IN_MINUTES", "EXPIRATION_IN_MINUTES")
	addEnvToViper("OPENAI_API_KEY", "OPENAI_API_KEY")
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

	AWSRegion           string `mapstructure:"AWS_REGION"`
	AWSS3Bucket         string `mapstructure:"AWS_S3_BUCKET"`
	AWSAccessKey        string `mapstructure:"AWS_ACCESS_KEY"`
	AWSSecretKey        string `mapstructure:"AWS_SECRET_KEY"`
	AWSEndpoint         string `mapstructure:"AWS_ENDPOINT"`
	ExpirationInMinutes int    `mapstructure:"EXPIRATION_IN_MINUTES"`

	OpenAIAPIKey string `mapstructure:"OPENAI_API_KEY"`

	S3Config        storage.S3Config `mapstructure:",squash"`
	DBConfig        db.Config        `mapstructure:",squash"`
	RedisConfig     redis.Config     `mapstructure:",squash"`
	TelemetryConfig telemetry.Config `mapstructure:",squash"`
}

// NewAppConfig 是一個 Provider，供 Wire 依賴注入使用
func NewAppConfig() (*AppConfig, error) {
	var cfg AppConfig

	err := LoadConfig(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
