package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Region              string `mapstructure:"AWS_REGION"`
	Bucket              string `mapstructure:"AWS_S3_BUCKET"`
	AccessKey           string `mapstructure:"AWS_ACCESS_KEY"`
	SecretKey           string `mapstructure:"AWS_SECRET_KEY"`
	Endpoint            string `mapstructure:"AWS_ENDPOINT"`
	ExpirationInMinutes int    `mapstructure:"EXPIRATION_IN_MINUTES"`
}

type S3Storage struct {
	client              *s3.Client
	bucket              string
	expirationInMinutes int
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	client := s3.NewFromConfig(aws.Config{
		Region: cfg.Region,
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey,
			}, nil
		}),
	})

	if client == nil {
		return nil, fmt.Errorf("failed to create S3 client")
	}

	return &S3Storage{
		client:              client,
		bucket:              cfg.Bucket,
		expirationInMinutes: cfg.ExpirationInMinutes,
	}, nil
}

func (s *S3Storage) GenerateUploadURL(ctx context.Context, objectKey, contentType string) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}

	presignedURL, err := presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(s.expirationInMinutes) * time.Minute
	})
	if err != nil {
		return "", err
	}

	return presignedURL.URL, nil
}

func (s *S3Storage) DownloadToTempFile(ctx context.Context, s3Key string) (string, error) {
	file, err := os.CreateTemp("", "stt-audio-*.tmp")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			os.Remove(file.Name())
		}
	}()

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	return file.Name(), nil
}
