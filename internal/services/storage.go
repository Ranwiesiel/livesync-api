package services

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/Ranwiesiel/livesync-api/internal/config"
)

// R2Storage membungkus client S3 yang sudah diarahkan ke Cloudflare R2.
type R2Storage struct {
	Client *s3.Client
}

func NewR2(ctx context.Context, cfg appconfig.StorageConfig) (*R2Storage, error) {
	if cfg.S3APIEndpoint == "" {
		return nil, fmt.Errorf("storage: S3_API_ENDPOINT belum diatur")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("storage: ACCESS_KEY_ID / SECRET_ACCESS_KEY belum diatur")
	}

	client := s3.New(s3.Options{
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(cfg.S3APIEndpoint),
		UsePathStyle: true,
	})

	return &R2Storage{Client: client}, nil
}

// ListBuckets mengembalikan nama bucket yang bisa diakses kredensial ini.
// Dipakai untuk memastikan koneksi dan izin benar-benar bekerja.
func (r *R2Storage) ListBuckets(ctx context.Context) ([]string, error) {
	out, err := r.Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("storage: list buckets: %w", err)
	}

	names := make([]string, 0, len(out.Buckets))
	for _, b := range out.Buckets {
		names = append(names, aws.ToString(b.Name))
	}
	return names, nil
}
