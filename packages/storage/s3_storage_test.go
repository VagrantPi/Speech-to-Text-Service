package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewS3Storage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := S3Config{
			Region:              "us-east-1",
			Bucket:              "test-bucket",
			AccessKey:           "test-key",
			SecretKey:           "test-secret",
			Endpoint:            "",
			PublicEndpoint:      "",
			ExpirationInMinutes: 15,
		}

		storage, err := NewS3Storage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, "test-bucket", storage.bucket)
		assert.Equal(t, 15, storage.expirationInMinutes)
	})

	t.Run("with endpoint", func(t *testing.T) {
		cfg := S3Config{
			Region:              "us-east-1",
			Bucket:              "test-bucket",
			AccessKey:           "test-key",
			SecretKey:           "test-secret",
			Endpoint:            "http://localhost:9000",
			PublicEndpoint:      "",
			ExpirationInMinutes: 15,
		}

		storage, err := NewS3Storage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
	})

	t.Run("with public endpoint", func(t *testing.T) {
		cfg := S3Config{
			Region:              "us-east-1",
			Bucket:              "test-bucket",
			AccessKey:           "test-key",
			SecretKey:           "test-secret",
			Endpoint:            "http://localhost:9000",
			PublicEndpoint:      "http://public.example.com",
			ExpirationInMinutes: 15,
		}

		storage, err := NewS3Storage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.NotNil(t, storage.publicClient)
	})

	t.Run("without public endpoint", func(t *testing.T) {
		cfg := S3Config{
			Region:              "us-east-1",
			Bucket:              "test-bucket",
			AccessKey:           "test-key",
			SecretKey:           "test-secret",
			Endpoint:            "http://localhost:9000",
			PublicEndpoint:      "",
			ExpirationInMinutes: 15,
		}

		storage, err := NewS3Storage(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Nil(t, storage.publicClient)
	})
}

func TestS3Storage_Fields(t *testing.T) {
	cfg := S3Config{
		Region:              "ap-northeast-1",
		Bucket:              "my-bucket",
		AccessKey:           "AKIAIOSFODNN7EXAMPLE",
		SecretKey:           "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Endpoint:            "http://minio:9000",
		PublicEndpoint:      "https://cdn.example.com",
		ExpirationInMinutes: 60,
	}

	storage, err := NewS3Storage(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, storage.client)
	assert.Equal(t, "my-bucket", storage.bucket)
	assert.Equal(t, 60, storage.expirationInMinutes)
	assert.Equal(t, "https://cdn.example.com", storage.publicEndpoint)
	assert.NotNil(t, storage.publicClient)
}

func TestS3Storage_GenerateUploadURL(t *testing.T) {
	cfg := S3Config{
		Region:              "us-east-1",
		Bucket:              "test-bucket",
		AccessKey:           "test-key",
		SecretKey:           "test-secret",
		ExpirationInMinutes: 15,
	}

	storage, err := NewS3Storage(cfg)
	assert.NoError(t, err)

	url, err := storage.GenerateUploadURL(context.Background(), "audio/test.wav", "audio/wav")
	assert.NoError(t, err)
	assert.Contains(t, url, "audio/test.wav")
	assert.Contains(t, url, "test-bucket")
}

func TestS3Storage_GenerateUploadURL_WithPublicClient(t *testing.T) {
	cfg := S3Config{
		Region:              "us-east-1",
		Bucket:              "test-bucket",
		AccessKey:           "test-key",
		SecretKey:           "test-secret",
		Endpoint:            "http://localhost:9000",
		PublicEndpoint:      "http://public.example.com",
		ExpirationInMinutes: 15,
	}

	storage, err := NewS3Storage(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, storage.publicClient)

	url, err := storage.GenerateUploadURL(context.Background(), "audio/test.wav", "audio/wav")
	assert.NoError(t, err)
	assert.Contains(t, url, "audio/test.wav")
}

func TestS3Config_MapStructureTags(t *testing.T) {
	cfg := S3Config{
		Region:              "us-east-1",
		Bucket:              "test-bucket",
		AccessKey:           "access-key",
		SecretKey:           "secret-key",
		Endpoint:            "http://localhost:9000",
		PublicEndpoint:      "http://public.example.com",
		ExpirationInMinutes: 30,
	}

	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "test-bucket", cfg.Bucket)
	assert.Equal(t, "access-key", cfg.AccessKey)
	assert.Equal(t, "secret-key", cfg.SecretKey)
	assert.Equal(t, "http://localhost:9000", cfg.Endpoint)
	assert.Equal(t, "http://public.example.com", cfg.PublicEndpoint)
	assert.Equal(t, 30, cfg.ExpirationInMinutes)
}

func TestS3Storage_BucketField(t *testing.T) {
	tests := []struct {
		name        string
		bucket      string
		expectEmpty bool
	}{
		{"normal bucket", "my-audio-bucket", false},
		{"empty bucket", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := S3Config{
				Region:              "us-east-1",
				Bucket:              tt.bucket,
				AccessKey:           "test-key",
				SecretKey:           "test-secret",
				ExpirationInMinutes: 15,
			}

			storage, err := NewS3Storage(cfg)
			assert.NoError(t, err)

			if tt.expectEmpty {
				assert.Empty(t, storage.bucket)
			} else {
				assert.Equal(t, tt.bucket, storage.bucket)
			}
		})
	}
}

func TestS3Storage_ExpirationInMinutes(t *testing.T) {
	tests := []struct {
		name    string
		minutes int
	}{
		{"default 15 minutes", 15},
		{"60 minutes", 60},
		{"5 minutes", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := S3Config{
				Region:              "us-east-1",
				Bucket:              "test-bucket",
				AccessKey:           "test-key",
				SecretKey:           "test-secret",
				ExpirationInMinutes: tt.minutes,
			}

			storage, err := NewS3Storage(cfg)
			assert.NoError(t, err)
			assert.Equal(t, tt.minutes, storage.expirationInMinutes)
		})
	}
}