package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes membuat index yang dibutuhkan query aplikasi. Aman dipanggil
// berulang kali: MongoDB mengabaikan index yang sudah ada dengan definisi sama.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	photos := db.Collection("photos")

	// Pagination memakai filter event_id lalu sort _id menurun, jadi kedua
	// field harus ada di satu index komposit. Tanpa ini, query harus memindai
	// seluruh koleksi saat foto bertambah banyak.
	_, err := photos.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "event_id", Value: 1},
			{Key: "_id", Value: -1},
		},
		Options: options.Index().SetName("event_id_1__id_-1"),
	})
	if err != nil {
		return fmt.Errorf("database: buat index photos: %w", err)
	}

	return nil
}
