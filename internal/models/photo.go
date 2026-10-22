package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Photo struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID    bson.ObjectID `bson:"event_id" json:"event_id"`
	StorageKey string        `bson:"storage_key" json:"storage_key"`
	URL        string        `bson:"url" json:"url"`
	BlurHash   string        `bson:"blur_hash" json:"blur_hash"`
	Uploader   string        `bson:"uploader" json:"uploader"`
	CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
}
