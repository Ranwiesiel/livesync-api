package main

import (
	"context"
	"log"
	"time"

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

	store, err := services.NewR2(ctx, cfg)
	if err != nil {
		log.Fatal("Gagal menginisialisasi R2: ", err)
	}

	photoHandler := handlers.NewPhotoHandler(
		db.Collection("photos"),
		db.Collection("events"),
		store,
	)

	names, err := store.ListBuckets(ctx)
	if err != nil {
		log.Fatal("Gagal mengakses R2: ", err)
	}
	log.Println("R2 siap, bucket:", names)

	photos := db.Collection("photos")

	processor := worker.New(store, func(ctx context.Context, id bson.ObjectID, fields map[string]any) error {
		_, err := photos.UpdateByID(ctx, id, bson.M{"$set": fields})
		return err
	})
	processor.Run(context.Background(), 4)
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
