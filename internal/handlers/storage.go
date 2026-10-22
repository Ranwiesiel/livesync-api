package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Ranwiesiel/livesync-api/internal/response"
	"github.com/Ranwiesiel/livesync-api/internal/services"
)

// presignTTL adalah masa berlaku URL upload.
const presignTTL = 15 * time.Minute

// allowedContentTypes membatasi apa yang boleh di-upload ke bucket.
var allowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
	"image/heic": true,
	"image/heif": true,
}

// StorageHandler melayani endpoint yang berhubungan dengan Cloudflare R2.
type StorageHandler struct {
	storage *services.R2Storage
}

func NewStorageHandler(storage *services.R2Storage) *StorageHandler {
	return &StorageHandler{storage: storage}
}

type PresignUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
}

type PresignUploadResponse struct {
	UploadURL   string    `json:"upload_url"`
	Method      string    `json:"method"`
	Key         string    `json:"key"`
	ContentType string    `json:"content_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// POST /api/media/presign-upload
//
// Menerima nama file dan tipe MIME, lalu mengembalikan URL PUT yang berlaku
// selama 15 menit. Client meng-upload langsung ke R2, bukan lewat server ini.
func (h *StorageHandler) PresignUpload(c *gin.Context) {
	var req PresignUploadRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Failure(c, http.StatusBadRequest, "Request body tidak valid")
		return
	}

	filename := strings.TrimSpace(req.Filename)
	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))

	if filename == "" || contentType == "" {
		response.Failure(c, http.StatusBadRequest, "Filename dan content_type wajib diisi")
		return
	}

	if !allowedContentTypes[contentType] {
		response.Failure(c, http.StatusUnsupportedMediaType, "Tipe file tidak didukung")
		return
	}

	key := buildObjectKey(filename)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	upload, err := h.storage.PresignPut(ctx, key, contentType, presignTTL)
	if err != nil {
		response.Failure(c, http.StatusInternalServerError, "Gagal membuat URL upload")
		return
	}

	response.Success(c, http.StatusOK, PresignUploadResponse{
		UploadURL:   upload.URL,
		Method:      upload.Method,
		Key:         upload.Key,
		ContentType: contentType,
		ExpiresIn:   int(presignTTL.Seconds()),
		ExpiresAt:   upload.ExpiresAt,
	})
}

// RegisterStorageRoutes mendaftarkan seluruh route storage ke router.
func RegisterStorageRoutes(router *gin.Engine, storage *services.R2Storage) {
	handler := NewStorageHandler(storage)

	group := router.Group("/api/media")
	group.POST("/presign-upload", handler.PresignUpload)
}

// buildObjectKey menghasilkan key yang aman dan unik, misalnya
// "uploads/2026/10/22/9f3a1c2b-foto.jpg". Nama dari client tidak dipercaya
// begitu saja: path dipangkas dan karakter aneh diganti.
func buildObjectKey(filename string) string {
	name := filepath.Base(filename)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	name = strings.TrimLeft(name, ".")
	if name == "" {
		name = "file"
	}
	if len(name) > 80 {
		name = name[len(name)-80:]
	}

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		// crypto/rand gagal berarti ada masalah serius di sistem.
		panic(fmt.Sprintf("storage: gagal membuat nama acak: %v", err))
	}

	now := time.Now().UTC()
	return fmt.Sprintf("uploads/%d/%02d/%02d/%s-%s",
		now.Year(), now.Month(), now.Day(), hex.EncodeToString(suffix), name)
}
