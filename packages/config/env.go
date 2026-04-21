package config

import (
	"log"

	"github.com/spf13/viper"
	"speech.local/packages/db"
	"speech.local/packages/storage"
)

// LoadConfig 負責將 .env 或系統環境變數載入到指定的 Struct 中
// path 通常傳入 "." (根目錄) 或 "../.." (視執行檔位置而定)
func LoadConfig(configStruct interface{}) error {
	viper.AddConfigPath(".")
	viper.AddConfigPath("../..")
	viper.SetConfigName(".env")
	viper.SetConfigType("env") // 即使檔名沒有副檔名，也當作 env 格式解析

	// 自動讀取系統環境變數 (重要：會覆蓋 .env 的設定)
	viper.AutomaticEnv()

	// 嘗試讀取檔案
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("警告: 找不到 .env 檔案，將完全依賴系統環境變數")
		} else {
			return err
		}
	}

	// 將讀取到的設定反序列化到傳入的 struct (必須是指標)
	err := viper.Unmarshal(configStruct)
	return err
}

// AppConfig 是 API Server 專屬的設定總表
// 使用 mapstructure tag 來對應 Viper 讀取出來的 key
type AppConfig struct {
	APIPort   string `mapstructure:"API_PORT"`
	RedisHost string `mapstructure:"REDIS_HOST"`
	MQURL     string `mapstructure:"MQ_URL"`

	AWSRegion           string `mapstructure:"AWS_REGION"`
	AWSS3Bucket         string `mapstructure:"AWS_S3_BUCKET"`
	AWSAccessKey        string `mapstructure:"AWS_ACCESS_KEY"`
	AWSSecretKey        string `mapstructure:"AWS_SECRET_KEY"`
	AWSEndpoint         string `mapstructure:"AWS_ENDPOINT"`
	ExpirationInMinutes int    `mapstructure:"EXPIRATION_IN_MINUTES"`

	S3Config storage.S3Config `mapstructure:",squash"`

	DBConfig db.Config `mapstructure:",squash"`
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
