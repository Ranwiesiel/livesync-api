package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newTestStorage() *R2Storage {
	return &R2Storage{
		Client: s3.New(s3.Options{
			Region:       "auto",
			Credentials:  credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
			BaseEndpoint: aws.String("https://contoh.r2.cloudflarestorage.com"),
			UsePathStyle: true,
		}),
		Bucket: "livesync-test",
	}
}

func TestPresignGet(t *testing.T) {
	store := newTestStorage()

	url, err := store.PresignGet(context.Background(),
		"uploads/2026/10/22/foto.png", time.Hour)
	if err != nil {
		t.Fatalf("PresignGet() error = %v", err)
	}

	for _, want := range []string{
		"livesync-test",
		"uploads/2026/10/22/foto.png",
		"X-Amz-Signature=",
		"X-Amz-Expires=3600",
	} {
		if !strings.Contains(url, want) {
			t.Errorf("PresignGet() = %q, tidak mengandung %q", url, want)
		}
	}
}
