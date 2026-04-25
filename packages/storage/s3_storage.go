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
	PublicEndpoint      string `mapstructure:"AWS_PUBLIC_ENDPOINT"`
	ExpirationInMinutes int    `mapstructure:"EXPIRATION_IN_MINUTES"`
}

type S3Storage struct {
	client              *s3.Client
	bucket              string
	expirationInMinutes int
	publicEndpoint      string
	publicClient        *s3.Client
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	awsConfig := aws.Config{
		Region: cfg.Region,
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKey,
				SecretAccessKey: cfg.SecretKey,
			}, nil
		}),
	}

	if cfg.Endpoint != "" {
		awsConfig.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})

	if client == nil {
		return nil, fmt.Errorf("failed to create S3 client")
	}

	s := &S3Storage{
		client:              client,
		bucket:              cfg.Bucket,
		expirationInMinutes: cfg.ExpirationInMinutes,
		publicEndpoint:      cfg.PublicEndpoint,
	}

	if cfg.PublicEndpoint != "" {
		publicConfig := aws.Config{
			Region: cfg.Region,
			Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     cfg.AccessKey,
					SecretAccessKey: cfg.SecretKey,
				}, nil
			}),
			BaseEndpoint: aws.String(cfg.PublicEndpoint),
		}

		s.publicClient = s3.NewFromConfig(publicConfig, func(o *s3.Options) {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String(cfg.PublicEndpoint)
		})
	}

	return s, nil
}

func (s *S3Storage) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		return nil
	}
	_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if createErr != nil {
		return createErr
	}
	return nil
}

func (s *S3Storage) GenerateUploadURL(ctx context.Context, objectKey, contentType string) (string, error) {
	client := s.client
	if s.publicClient != nil {
		client = s.publicClient
	}
	presignClient := s3.NewPresignClient(client)

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
