package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Photo struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID      bson.ObjectID `bson:"event_id" json:"event_id"`
	StorageKey   string        `bson:"storage_key" json:"storage_key"`
	URL          string        `bson:"url" json:"url"`
	ThumbnailURL string        `bson:"thumbnail_url,omitempty" json:"thumbnail_url,omitempty"`
	Width        int           `bson:"width,omitempty" json:"width,omitempty"`
	Height       int           `bson:"height,omitempty" json:"height,omitempty"`
	BlurHash     string        `bson:"blur_hash,omitempty" json:"blur_hash,omitempty"`
	Uploader     string        `bson:"uploader" json:"uploader"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
}