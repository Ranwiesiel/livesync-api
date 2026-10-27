package main

import (
	"context"
	"log"
	"time"

	"github.com/Ranwiesiel/livesync-api/internal/config"
	"github.com/Ranwiesiel/livesync-api/internal/database"
	"github.com/Ranwiesiel/livesync-api/internal/handlers"
	"github.com/Ranwiesiel/livesync-api/internal/router"
	"github.com/Ranwiesiel/livesync-api/internal/services"
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
