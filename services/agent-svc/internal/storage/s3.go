package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the AWS S3 client for MinIO/R2 compatibility.
type S3Client struct {
	client *s3.Client
	bucket string
}

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

func NewS3Client(ctx context.Context, cfg S3Config) (*S3Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			if cfg.Endpoint != "" {
				return aws.Endpoint{
					URL:               cfg.Endpoint,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		},
	)

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO
	})

	return &S3Client{client: client, bucket: cfg.Bucket}, nil
}

// GetImageBase64 fetches an image from S3 and returns it as base64 + MIME type.
func (c *S3Client) GetImageBase64(ctx context.Context, key string) (string, string, error) {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", "", fmt.Errorf("s3 get object %q: %w", key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return "", "", fmt.Errorf("read s3 body: %w", err)
	}

	mimeType := "image/jpeg"
	if out.ContentType != nil && *out.ContentType != "" {
		mimeType = *out.ContentType
	}

	return base64.StdEncoding.EncodeToString(data), mimeType, nil
}

// PutObject uploads data to S3 and returns the object key.
func (c *S3Client) PutObject(ctx context.Context, key, contentType string, data []byte) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

// PresignedURL generates a pre-signed GET URL valid for the given duration.
func (c *S3Client) PresignedURL(ctx context.Context, key string) (string, error) {
	presign := s3.NewPresignClient(c.client)
	req, err := presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
