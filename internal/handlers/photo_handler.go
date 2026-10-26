
package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Ranwiesiel/livesync-api/internal/models"
	"github.com/Ranwiesiel/livesync-api/internal/response"
)

type PhotoHandler struct {
	photos *mongo.Collection
	events *mongo.Collection
}

func NewPhotoHandler(
	photos *mongo.Collection,
	events *mongo.Collection,
) *PhotoHandler {
	return &PhotoHandler{
		photos: photos,
		events: events,
	}
}

type ConfirmPhotoRequest struct {
	EventID    string `json:"event_id" binding:"required"`
	StorageKey string `json:"storage_key" binding:"required"`
	URL        string `json:"url" binding:"required"`
	BlurHash   string `json:"blur_hash"`
	Uploader   string `json:"uploader"`
}

// POST /api/media/confirm
func (h *PhotoHandler) ConfirmPhoto(c *gin.Context) {
	var req ConfirmPhotoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Failure(c, http.StatusBadRequest, "Request body tidak valid")
		return
	}

	req.EventID = strings.TrimSpace(req.EventID)
	req.StorageKey = strings.TrimSpace(req.StorageKey)
	req.URL = strings.TrimSpace(req.URL)

	if req.EventID == "" || req.StorageKey == "" || req.URL == "" {
		response.Failure(c, http.StatusBadRequest,
			"Event ID, storage key, dan URL wajib diisi")
		return
	}

	eventID, err := bson.ObjectIDFromHex(req.EventID)
	if err != nil {
		response.Failure(c, http.StatusBadRequest, "Event ID tidak valid")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var event bson.M
	err = h.events.FindOne(ctx, bson.D{
		{Key: "_id", Value: eventID},
	}).Decode(&event)

	if err == mongo.ErrNoDocuments {
		response.Failure(c, http.StatusNotFound, "Event tidak ditemukan")
		return
	}
	if err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal memeriksa event")
		return
	}

	photo := models.Photo{
		ID:         bson.NewObjectID(),
		EventID:    eventID,
		StorageKey: req.StorageKey,
		URL:        req.URL,
		BlurHash:   req.BlurHash,
		Uploader:   req.Uploader,
		CreatedAt:  time.Now().UTC(),
	}

	if _, err := h.photos.InsertOne(ctx, photo); err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal menyimpan metadata foto")
		return
	}

	response.Success(c, http.StatusCreated, photo)
}

// GET /api/events/:id/photos
func (h *PhotoHandler) GetEventPhotos(c *gin.Context) {
	eventID, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Failure(c, http.StatusBadRequest, "Event ID tidak valid")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Memastikan event memang ada.
	err = h.events.FindOne(ctx, bson.D{
		{Key: "_id", Value: eventID},
	}).Err()

	if err == mongo.ErrNoDocuments {
		response.Failure(c, http.StatusNotFound, "Event tidak ditemukan")
		return
	}
	if err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal memeriksa event")
		return
	}

	cursor, err := h.photos.Find(
		ctx,
		bson.D{{Key: "event_id", Value: eventID}},
		options.Find().SetSort(bson.D{
			{Key: "created_at", Value: -1},
		}),
	)
	if err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal mengambil foto event")
		return
	}
	defer cursor.Close(ctx)

	photos := make([]models.Photo, 0)
	if err := cursor.All(ctx, &photos); err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal membaca data foto")
		return
	}

	response.Success(c, http.StatusOK, photos)
}
