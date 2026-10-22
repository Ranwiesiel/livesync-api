package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/Ranwiesiel/livesync-api/internal/config"
)

// R2Storage membungkus client S3 yang sudah diarahkan ke Cloudflare R2.
type R2Storage struct {
	Client *s3.Client
	Bucket string
}

// NewR2 membuat client S3 untuk Cloudflare R2.
//
// Tiga hal membedakannya dari client S3 biasa:
//   - region harus "auto" (R2 tidak punya region seperti AWS)
//   - BaseEndpoint diarahkan ke endpoint R2 milik account
//   - UsePathStyle aktif, karena R2 memakai gaya path, bukan virtual-host
func NewR2(ctx context.Context, cfg appconfig.Config) (*R2Storage, error) {
	if cfg.S3APIEndpoint == "" {
		return nil, fmt.Errorf("storage: S3_API_ENDPOINT belum diatur")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("storage: ACCESS_KEY_ID / SECRET_ACCESS_KEY belum diatur")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("storage: R2_BUCKET belum diatur")
	}

	client := s3.New(s3.Options{
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(cfg.S3APIEndpoint),
		UsePathStyle: true,
	})

	return &R2Storage{Client: client, Bucket: cfg.Bucket}, nil
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

// PresignedUpload adalah instruksi upload yang dikirim ke client. Client hanya
// perlu melakukan PUT ke URL dengan header Content-Type yang sama persis
// seperti saat URL ditandatangani.
type PresignedUpload struct {
	URL       string    `json:"upload_url"`
	Method    string    `json:"method"`
	Key       string    `json:"key"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PresignPut menghasilkan URL PUT sementara untuk satu object di bucket.
//
// contentType ikut ditandatangani, jadi client wajib mengirim header
// Content-Type yang sama. Kalau berbeda, R2 menolak dengan SignatureDoesNotMatch.
func (r *R2Storage) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (PresignedUpload, error) {
	presigner := s3.NewPresignClient(r.Client)

	req, err := presigner.PresignPutObject(ctx,
		&s3.PutObjectInput{
			Bucket:      aws.String(r.Bucket),
			Key:         aws.String(key),
			ContentType: aws.String(contentType),
		},
		func(o *s3.PresignOptions) {
			o.Expires = ttl
		},
	)
	if err != nil {
		return PresignedUpload{}, fmt.Errorf("storage: presign put: %w", err)
	}

	return PresignedUpload{
		URL:       req.URL,
		Method:    req.Method,
		Key:       key,
		ExpiresAt: time.Now().UTC().Add(ttl),
	}, nil
}
