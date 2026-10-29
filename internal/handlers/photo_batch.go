package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/Ranwiesiel/livesync-api/internal/models"
	"github.com/Ranwiesiel/livesync-api/internal/response"
	"github.com/Ranwiesiel/livesync-api/internal/worker"
)

// maxBatchPhotos membatasi jumlah foto per request supaya satu batch tidak
// menghabiskan waktu request dan memori server.
const maxBatchPhotos = 50

type BatchPhotoItem struct {
	StorageKey string `json:"storage_key"`
	BlurHash   string `json:"blur_hash"`
}

type ConfirmPhotosBatchRequest struct {
	EventID  string           `json:"event_id"`
	Uploader string           `json:"uploader"`
	Photos   []BatchPhotoItem `json:"photos"`
}

type BatchPhotoFailure struct {
	StorageKey string `json:"storage_key"`
	Message    string `json:"message"`
}

type ConfirmPhotosBatchResponse struct {
	Items  []models.Photo      `json:"items"`
	Failed []BatchPhotoFailure `json:"failed"`
}

// batchPlan adalah hasil validasi request: item yang siap disimpan dan item
// yang sudah pasti gagal. Item gagal tidak membatalkan seluruh batch.
type batchPlan struct {
	EventID  string
	Uploader string
	Items    []BatchPhotoItem
	Failed   []BatchPhotoFailure
}

func planBatch(req ConfirmPhotosBatchRequest) (batchPlan, error) {
	plan := batchPlan{
		EventID:  strings.TrimSpace(req.EventID),
		Uploader: strings.TrimSpace(req.Uploader),
		Items:    make([]BatchPhotoItem, 0, len(req.Photos)),
		Failed:   make([]BatchPhotoFailure, 0),
	}

	if plan.EventID == "" {
		return batchPlan{}, errors.New("event_id wajib diisi")
	}
	if len(req.Photos) == 0 {
		return batchPlan{}, errors.New("photos wajib berisi minimal satu file")
	}
	if len(req.Photos) > maxBatchPhotos {
		return batchPlan{}, fmt.Errorf("maksimal %d foto per batch", maxBatchPhotos)
	}

	seen := make(map[string]struct{}, len(req.Photos))
	for _, item := range req.Photos {
		key := strings.TrimSpace(item.StorageKey)

		if _, duplicate := seen[key]; duplicate && key != "" {
			continue
		}

		if err := validateStorageKey(key); err != nil {
			plan.Failed = append(plan.Failed, BatchPhotoFailure{
				StorageKey: key,
				Message:    "storage key tidak valid",
			})
			continue
		}

		seen[key] = struct{}{}
		plan.Items = append(plan.Items, BatchPhotoItem{
			StorageKey: key,
			BlurHash:   item.BlurHash,
		})
	}

	return plan, nil
}

// POST /api/media/confirm-batch
func (h *PhotoHandler) ConfirmBatch(c *gin.Context) {
	var req ConfirmPhotosBatchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Failure(c, http.StatusBadRequest, "Request body tidak valid")
		return
	}

	plan, err := planBatch(req)
	if err != nil {
		response.Failure(c, http.StatusBadRequest, err.Error())
		return
	}

	eventID, err := bson.ObjectIDFromHex(plan.EventID)
	if err != nil {
		response.Failure(c, http.StatusBadRequest, "Event ID tidak valid")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	err = h.events.FindOne(ctx, bson.D{
		{Key: "_id", Value: eventID},
	}).Err()

	if errors.Is(err, mongo.ErrNoDocuments) {
		response.Failure(c, http.StatusNotFound, "Event tidak ditemukan")
		return
	}
	if err != nil {
		response.Failure(c, http.StatusInternalServerError, "Gagal memeriksa event")
		return
	}

	result := ConfirmPhotosBatchResponse{
		Items:  make([]models.Photo, 0, len(plan.Items)),
		Failed: plan.Failed,
	}

	for _, item := range plan.Items {
		photo, err := h.confirmBatchItem(ctx, eventID, plan.Uploader, item)
		if err != nil {
			result.Failed = append(result.Failed, BatchPhotoFailure{
				StorageKey: item.StorageKey,
				Message:    err.Error(),
			})
			continue
		}

		result.Items = append(result.Items, photo)
	}

	response.Success(c, http.StatusOK, result)
}

// confirmBatchItem menyimpan metadata satu foto. Konfirmasi ulang untuk
// storage_key yang sama mengembalikan dokumen lama, jadi retry tidak membuat
// duplikat.
func (h *PhotoHandler) confirmBatchItem(
	ctx context.Context,
	eventID bson.ObjectID,
	uploader string,
	item BatchPhotoItem,
) (models.Photo, error) {
	signedURL, err := h.storage.PresignGet(ctx, item.StorageKey, photoURLTTL)
	if err != nil {
		return models.Photo{}, errors.New("gagal menyiapkan URL foto")
	}

	var existing models.Photo
	err = h.photos.FindOne(ctx, bson.D{
		{Key: "storage_key", Value: item.StorageKey},
	}).Decode(&existing)

	if err == nil {
		existing.URL = signedURL
		return existing, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return models.Photo{}, errors.New("gagal memeriksa metadata foto")
	}

	photo := models.Photo{
		ID:         bson.NewObjectID(),
		EventID:    eventID,
		StorageKey: item.StorageKey,
		URL:        signedURL,
		BlurHash:   item.BlurHash,
		Uploader:   uploader,
		CreatedAt:  time.Now().UTC(),
	}

	if _, err := h.photos.InsertOne(ctx, photo); err != nil {
		return models.Photo{}, errors.New("gagal menyimpan metadata foto")
	}

	if !worker.Enqueue(worker.Job{PhotoID: photo.ID, Key: photo.StorageKey}) {
		log.Printf("antrean penuh, foto %s belum diproses", photo.ID.Hex())
	}

	return photo, nil
}
