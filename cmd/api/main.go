
package main

import (
	"context"
	"log"
	"time"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/Ranwiesiel/livesync-api/internal/config"
	"github.com/Ranwiesiel/livesync-api/internal/database"
	"github.com/Ranwiesiel/livesync-api/internal/handlers"
	"github.com/Ranwiesiel/livesync-api/internal/router"
	"github.com/Ranwiesiel/livesync-api/internal/services"
	"github.com/Ranwiesiel/livesync-api/internal/worker"
)

func main() {
	cfg := config.LoadEnv()

	client, err := database.GetMongoClient()
	if err != nil {
		log.Fatal("Gagal menginisialisasi MongoDB: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := client.Database(cfg.MongoDatabase)
	log.Println("Database siap:", db.Name())

	if err := database.EnsureIndexes(ctx, db); err != nil {
		log.Fatal("Gagal membuat index: ", err)
	}
	log.Println("Index database siap")

	store, err := services.NewR2(ctx, cfg)
	if err != nil {
		log.Fatal("Gagal menginisialisasi R2: ", err)
	}

	photos := db.Collection("photos")

	photoHandler := handlers.NewPhotoHandler(
		photos,
		db.Collection("events"),
		store,
	)

	names, err := store.ListBuckets(ctx)
	if err != nil {
		log.Fatal("Gagal mengakses R2: ", err)
	}
	log.Println("R2 siap, bucket:", names)

	// Updater worker: memperbarui field tertentu tanpa mengganti
	// seluruh dokumen foto.
	updatePhoto := func(
		ctx context.Context,
		id bson.ObjectID,
		fields map[string]any,
	) error {
		result, err := photos.UpdateByID(
			ctx,
			id,
			bson.M{"$set": fields},
		)
		if err != nil {
			return err
		}

		if result.MatchedCount == 0 {
			return fmt.Errorf("foto dengan ID %s tidak ditemukan", id.Hex())
		}

		return nil
	}

	processor := worker.New(store, updatePhoto)

	// Context terpisah agar worker tidak berhenti setelah context
	// inisialisasi 10 detik di atas kedaluwarsa.
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()

	processor.Run(workerCtx, 4)
	log.Println("Worker gambar berjalan: 4 goroutine")

	r := router.New(
		handlers.NewEventHandler(db.Collection("events")),
		handlers.NewStorageHandler(store),
		photoHandler,
		cfg.CORSOrigins,
	)

	log.Println("Server berjalan pada port:", cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Gagal menjalankan server: ", err)
	}
}
