
package handlers

import (
	"context"
	"errors"
	"fmt"
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

// objectKeyPrefix adalah folder tempat presign menaruh object. Confirm hanya
// menerima key dari folder ini supaya client tidak bisa mendaftarkan object
// sembarangan di bucket.
const objectKeyPrefix = "uploads/"

// photoURLTTL membatasi umur URL GET bertanda tangan. Dua puluh empat jam
// cukup untuk satu hari acara, termasuk foto lazy-load yang baru di-scroll
// beberapa jam setelah halaman dibuka.
const photoURLTTL = 24 * time.Hour

// photoURLSigner adalah bagian kecil dari storage yang dipakai handler foto.
// Dengan interface ini handler tidak perlu tahu detail S3/R2.
type photoURLSigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type PhotoHandler struct {
	photos  *mongo.Collection
	events  *mongo.Collection
	storage photoURLSigner
}

func NewPhotoHandler(
	photos *mongo.Collection,
	events *mongo.Collection,
	storage photoURLSigner,
) *PhotoHandler {
	return &PhotoHandler{
		photos:  photos,
		events:  events,
		storage: storage,
	}
}

type ConfirmPhotoRequest struct {
	EventID    string `json:"event_id" binding:"required"`
	StorageKey string `json:"storage_key" binding:"required"`
	BlurHash   string `json:"blur_hash"`
	Uploader   string `json:"uploader"`
}

// validateStorageKey memastikan key berasal dari folder upload dan tidak
// mencoba keluar dari folder tersebut.
func validateStorageKey(key string) error {
	key = strings.TrimSpace(key)

	if key == "" {
		return errors.New("storage key wajib diisi")
	}
	if !strings.HasPrefix(key, objectKeyPrefix) {
		return fmt.Errorf("storage key harus diawali %q", objectKeyPrefix)
	}
	if key == objectKeyPrefix || strings.HasSuffix(key, "/") {
		return errors.New("storage key tidak menunjuk ke file")
	}
	if strings.Contains(key, "..") ||
		strings.Contains(key, "//") ||
		strings.ContainsRune(key, 0) {
		return errors.New("storage key tidak valid")
	}

	return nil
}

// signPhotoURLs mengganti URL tersimpan dengan URL GET bertanda tangan yang
// berlaku sementara. URL publik bucket bisa saja tidak bisa diakses dari
// jaringan client; URL S3 bertanda tangan tetap bekerja.
func signPhotoURLs(
	ctx context.Context,
	signer photoURLSigner,
	ttl time.Duration,
	photos []models.Photo,
) error {
	for i := range photos {
		signed, err := signer.PresignGet(ctx, photos[i].StorageKey, ttl)
		if err != nil {
			return fmt.Errorf("menandatangani %s: %w", photos[i].StorageKey, err)
		}

		photos[i].URL = signed
	}

	return nil
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

	if req.EventID == "" || req.StorageKey == "" {
		response.Failure(c, http.StatusBadRequest,
			"Event ID dan storage key wajib diisi")
		return
	}

	if err := validateStorageKey(req.StorageKey); err != nil {
		response.Failure(c, http.StatusBadRequest, "Storage key tidak valid")
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

	signedURL, err := h.storage.PresignGet(ctx, req.StorageKey, photoURLTTL)
	if err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal menyiapkan URL foto")
		return
	}

	// Konfirmasi bisa terpanggil ulang (retry jaringan, tombol ditekan dua
	// kali). storage_key unik per upload, jadi dokumen lama dikembalikan
	// tanpa membuat duplikat.
	var existing models.Photo
	err = h.photos.FindOne(ctx, bson.D{
		{Key: "storage_key", Value: req.StorageKey},
	}).Decode(&existing)

	if err == nil {
		existing.URL = signedURL
		response.Success(c, http.StatusOK, existing)
		return
	}
	if err != mongo.ErrNoDocuments {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal memeriksa metadata foto")
		return
	}

	photo := models.Photo{
		ID:         bson.NewObjectID(),
		EventID:    eventID,
		StorageKey: req.StorageKey,
		URL:        signedURL,
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

	if err := signPhotoURLs(ctx, h.storage, photoURLTTL, photos); err != nil {
		response.Failure(c, http.StatusInternalServerError,
			"Gagal menyiapkan URL foto")
		return
	}

	response.Success(c, http.StatusOK, photos)
}
