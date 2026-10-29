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
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/Ranwiesiel/livesync-api/internal/database"
	"github.com/Ranwiesiel/livesync-api/internal/services"
)

func putSampleImage(t *testing.T, ctx context.Context, store *services.R2Storage, key string) {
	t.Helper()

	img := imaging.New(900, 600, color.NRGBA{R: 200, G: 80, B: 40, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := store.PutObject(ctx, key, buf.Bytes(), "image/jpeg"); err != nil {
		t.Fatalf("upload: %v", err)
	}
}

func TestJobQueueMemperbaruiDatabase(t *testing.T) {
	if os.Getenv("R2_TEST") == "" {
		t.Skip("set R2_TEST=1 untuk menjalankan test ini")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	store, err := services.NewR2(ctx, liveConfig(t))
	if err != nil {
		t.Fatalf("NewR2: %v", err)
	}

	client, err := database.GetMongoClient()
	if err != nil {
		t.Fatalf("Mongo: %v", err)
	}
	photos := client.Database("Livesync").Collection("photos")

	key := fmt.Sprintf("uploads/_test/queue-%d.jpg", time.Now().UnixNano())
	putSampleImage(t, ctx, store, key)
	defer store.DeleteObject(ctx, key)

	id := bson.NewObjectID()
	_, err = photos.InsertOne(ctx, bson.M{
		"_id":         id,
		"storage_key": key,
		"created_at":  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("InsertOne: %v", err)
	}
	defer photos.DeleteOne(ctx, bson.M{"_id": id})

	processor := New(store, func(ctx context.Context, id bson.ObjectID, fields map[string]any) error {
		_, err := photos.UpdateByID(ctx, id, bson.M{"$set": fields})
		return err
	})
	processor.Run(ctx, 2)

	if !Enqueue(Job{PhotoID: id, Key: key}) {
		t.Fatal("antrean penuh")
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var doc bson.M
		if err := photos.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
			t.Fatalf("FindOne: %v", err)
		}
		if url, ok := doc["thumbnail_url"].(string); ok && url != "" {
			hash, _ := doc["blur_hash"].(string)
			if hash == "" {
				t.Fatal("blur_hash tidak terisi")
			}
			t.Logf("thumbnail_url terisi, blur_hash=%s (%d char)", hash, len(hash))
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatal("worker tidak memperbarui dokumen dalam 30 detik")
}
