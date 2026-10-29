package models

import (
	"time"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CameraModelName menggabungkan produsen dan model kamera.
func (e *EXIFMetadata) CameraModelName() string {
	if e == nil {
		return ""
	}

	switch {
	case e.CameraMake == "":
		return e.CameraModel
	case e.CameraModel == "":
		return e.CameraMake
	case strings.Contains(
		strings.ToLower(e.CameraModel),
		strings.ToLower(e.CameraMake),
	):
		return e.CameraModel
	default:
		return fmt.Sprintf("%s %s", e.CameraMake, e.CameraModel)
	}
}

type EXIFMetadata struct {
    CameraMake       string   `bson:"camera_make,omitempty" json:"camera_make,omitempty"`
    CameraModel      string   `bson:"camera_model,omitempty" json:"camera_model,omitempty"`
    DateTimeOriginal string   `bson:"date_time_original,omitempty" json:"date_time_original,omitempty"`
    ISO              *int     `bson:"iso,omitempty" json:"iso,omitempty"`
    Aperture         *float64 `bson:"aperture,omitempty" json:"aperture,omitempty"`
    ShutterSpeed     string   `bson:"shutter_speed,omitempty" json:"shutter_speed,omitempty"`
}

type Photo struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID      bson.ObjectID `bson:"event_id" json:"event_id"`
	StorageKey   string        `bson:"storage_key" json:"storage_key"`
	URL          string        `bson:"-" json:"url"`
	ThumbnailURL string        `bson:"thumbnail_url,omitempty" json:"thumbnail_url,omitempty"`
	Width        int           `bson:"width,omitempty" json:"width,omitempty"`
	Height       int           `bson:"height,omitempty" json:"height,omitempty"`
	BlurHash     string        `bson:"blur_hash,omitempty" json:"blur_hash,omitempty"`
	Uploader     string        `bson:"uploader" json:"uploader"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	EXIF         *EXIFMetadata `bson:"exif,omitempty" json:"exif,omitempty"`
}