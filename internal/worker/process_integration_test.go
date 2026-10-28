package worker

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"github.com/disintegration/imaging"

	appconfig "github.com/Ranwiesiel/livesync-api/internal/config"
	"github.com/Ranwiesiel/livesync-api/internal/services"
)

func liveConfig(t *testing.T) appconfig.Config {
	t.Helper()

	cfg := appconfig.Config{
		AccessKeyID:     os.Getenv("ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("SECRET_ACCESS_KEY"),
		S3APIEndpoint:   os.Getenv("S3_API_ENDPOINT"),
		Bucket:          os.Getenv("R2_BUCKET"),
	}
	if cfg.Bucket == "" {
		cfg.Bucket = "livesync-dev"
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.S3APIEndpoint == "" {
		t.Skip("kredensial R2 tidak diatur")
	}
	return cfg
}

func TestProcessRoundTrip(t *testing.T) {
	if os.Getenv("R2_TEST") == "" {
		t.Skip("set R2_TEST=1 untuk menjalankan test ini")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, err := services.NewR2(ctx, liveConfig(t))
	if err != nil {
		t.Fatalf("NewR2: %v", err)
	}

	key := fmt.Sprintf("uploads/_test/%d.jpg", time.Now().UnixNano())

	original := imaging.New(1200, 800, color.NRGBA{R: 20, G: 120, B: 200, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, original, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode asli: %v", err)
	}
	if err := store.PutObject(ctx, key, buf.Bytes(), "image/jpeg"); err != nil {
		t.Fatalf("upload asli: %v", err)
	}
	defer store.DeleteObject(ctx, key)

	processor := New(store)
	res, err := processor.Process(ctx, key)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	defer store.DeleteObject(ctx, res.ThumbKey)

	thumb, contentType, err := store.GetObject(ctx, res.ThumbKey)
	if err != nil {
		t.Fatalf("thumbnail tidak ada di R2: %v", err)
	}
	if contentType != "image/jpeg" {
		t.Fatalf("content-type thumbnail = %q", contentType)
	}
	if len(thumb) != res.SizeBytes {
		t.Fatalf("ukuran thumbnail %d, result %d", len(thumb), res.SizeBytes)
	}

	t.Logf("key=%s thumb=%s blurhash=%s (%d char) %dx%d %d bytes",
		key, res.ThumbKey, res.BlurHash, len(res.BlurHash), res.Width, res.Height, res.SizeBytes)
}
