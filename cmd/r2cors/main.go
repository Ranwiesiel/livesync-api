package main

// Perintah kecil untuk memasang aturan CORS pada bucket R2.
//
// Browser meng-upload langsung ke endpoint R2, jadi bucket harus mengizinkan
// PUT dari origin frontend. Jalankan ulang kapan pun origin berubah:
//
//	go run ./cmd/r2cors
//
// Origin diambil dari CORS_ALLOWED_ORIGINS, sama seperti yang dipakai API.

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "github.com/Ranwiesiel/livesync-api/internal/config"
	"github.com/Ranwiesiel/livesync-api/internal/services"
)

func main() {
	cfg := appconfig.LoadEnv()

	store, err := services.NewR2(context.Background(), cfg)
	if err != nil {
		log.Fatal("Gagal menyiapkan R2: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rule := types.CORSRule{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "PUT", "HEAD"},
		AllowedHeaders: []string{"Content-Type"},
		ExposeHeaders:  []string{"ETag"},
		MaxAgeSeconds:  aws.Int32(3600),
	}

	_, err = store.Client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(store.Bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{rule},
		},
	})
	if err != nil {
		log.Fatalf("Gagal memasang CORS di bucket %s: %v", store.Bucket, err)
	}

	got, err := store.Client.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(store.Bucket),
	})
	if err != nil {
		log.Fatal("CORS terpasang tapi gagal dibaca ulang: ", err)
	}

	for _, applied := range got.CORSRules {
		log.Printf("bucket %s CORS: origins=[%s] methods=[%s] headers=[%s]",
			store.Bucket,
			strings.Join(applied.AllowedOrigins, " "),
			strings.Join(applied.AllowedMethods, " "),
			strings.Join(applied.AllowedHeaders, " "),
		)
	}
}
