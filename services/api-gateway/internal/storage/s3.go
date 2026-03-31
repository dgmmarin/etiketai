package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client    *s3.Client
	bucket    string
	publicURL string // externally reachable base URL (e.g. http://localhost:9000)
}

type Config struct {
	Endpoint  string
	PublicURL string // optional; falls back to Endpoint
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

func NewS3Client(ctx context.Context, cfg Config) (*S3Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	pub := cfg.PublicURL
	if pub == "" {
		pub = cfg.Endpoint
	}
	return &S3Client{client: client, bucket: cfg.Bucket, publicURL: pub}, nil
}

// ObjectURL returns the public HTTP URL for a stored object key.
// For local MinIO with anonymous read enabled, this is directly usable in <img src>.
func (c *S3Client) ObjectURL(key string) string {
	return strings.TrimRight(c.publicURL, "/") + "/" + c.bucket + "/" + key
}

// Upload buffers the reader and uploads to S3.
// Buffering is required because AWS SDK v2 needs a seekable body when using
// plain HTTP (no TLS) to compute checksums without trailing-checksum support.
func (c *S3Client) Upload(ctx context.Context, key, contentType string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(data))),
	})
	return err
}
