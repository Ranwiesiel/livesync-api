package main

import (
	"context"
	"log"
	"time"

	"github.com/Ranwiesiel/livesync-api/internal/config"
	"github.com/Ranwiesiel/livesync-api/internal/database"
	"github.com/Ranwiesiel/livesync-api/internal/services"
)

func main() {
	// LoadEnv membaca .env lebih dulu, supaya semua os.Getenv di bawahnya
	// (termasuk MONGODB_URI) sudah terisi.
	cfg := config.LoadEnv()

	client, err := database.GetMongoClient()
	if err != nil {
		log.Fatal("Gagal menginisialisasi MongoDB: ", err)
	}
	log.Println("Database siap:", client.Database("Livesync").Name())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Inisialisasi R2
	store, err := services.NewR2(ctx, cfg)
	if err != nil {
		log.Fatal("Gagal menginisialisasi R2: ", err)
	}

	names, err := store.ListBuckets(ctx)
	if err != nil {
		log.Fatal("Gagal mengakses R2: ", err)
	}
	log.Println("R2 siap, bucket:", names)
}
