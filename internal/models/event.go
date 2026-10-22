package models

import (
    "time"

    "go.mongodb.org/mongo-driver/v2/bson"
)

type Event struct {
    ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
    Title      string        `bson:"title" json:"title"`
    Code       string        `bson:"code" json:"code"`
    CoverImage string        `bson:"cover_image,omitempty" json:"cover_image"`
    CreatedAt  time.Time     `bson:"created_at" json:"created_at"`
    IsActive   bool          `bson:"is_active" json:"is_active"`
}
