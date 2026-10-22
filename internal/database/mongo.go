package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Ranwiesiel/livesync-api/internal/config"
)

var (
	client  *mongo.Client
	once    sync.Once
	initErr error
)

// GetMongoClient mengembalikan client MongoDB yang digunakan bersama.
func GetMongoClient() (*mongo.Client, error) {
	cfg := config.LoadEnv()
	once.Do(func() {
		uri := cfg.MongoDBURI
		if uri == "" {
			initErr = fmt.Errorf("MONGODB_URI belum diatur")
			return
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		client, initErr = mongo.Connect(
			options.Client().ApplyURI(uri),
		)
		if initErr != nil {
			return
		}

		if err := client.Ping(ctx, nil); err != nil {
			initErr = err
			_ = client.Disconnect(context.Background())
			client = nil
			return
		}
	})

	return client, initErr
}
