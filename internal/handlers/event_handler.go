
package handlers

import (
    "context"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"

    "github.com/Ranwiesiel/livesync-api/internal/models"
    "github.com/Ranwiesiel/livesync-api/internal/response"
)

type EventHandler struct {
    collection *mongo.Collection
}

func NewEventHandler(collection *mongo.Collection) *EventHandler {
    return &EventHandler{
        collection: collection,
    }
}

type CreateEventRequest struct {
    Title      string `json:"title" binding:"required"`
    Code       string `json:"code" binding:"required"`
    CoverImage string `json:"cover_image"`
}

// POST /api/events
func (h *EventHandler) CreateEvent(c *gin.Context) {
    var req CreateEventRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        response.Failure(c, 400, "Request body tidak valid")
        return
    }

    req.Title = strings.TrimSpace(req.Title)
    req.Code = strings.TrimSpace(req.Code)

    if req.Title == "" || req.Code == "" {
        response.Failure(c, 400, "Title dan code wajib diisi")
        return
    }

    ctx, cancel := context.WithTimeout(
        c.Request.Context(),
        5*time.Second,
    )
    defer cancel()

    var existing models.Event
    err := h.collection.FindOne(
        ctx,
        bson.D{{Key: "code", Value: req.Code}},
    ).Decode(&existing)

    if err == nil {
        response.Failure(c, 409, "Kode event sudah digunakan")
        return
    }

    if err != mongo.ErrNoDocuments {
        response.Failure(c, 500, "Gagal memeriksa event")
        return
    }

    event := models.Event{
        ID:         bson.NewObjectID(),
        Title:      req.Title,
        Code:       req.Code,
        CoverImage: req.CoverImage,
        CreatedAt:  time.Now().UTC(),
        IsActive:   true,
    }

    if _, err := h.collection.InsertOne(ctx, event); err != nil {
        response.Failure(c, 500, "Gagal membuat event")
        return
    }

    response.Success(c, 201, event)
}

// GET /api/events/:code
func (h *EventHandler) GetEvent(c *gin.Context) {
    code := strings.TrimSpace(c.Param("code"))

    if code == "" {
        response.Failure(c, 400, "Kode event wajib diisi")
        return
    }

    ctx, cancel := context.WithTimeout(
        c.Request.Context(),
        5*time.Second,
    )
    defer cancel()

    var event models.Event

    err := h.collection.FindOne(
        ctx,
        bson.D{{Key: "code", Value: code}},
    ).Decode(&event)

    if err == mongo.ErrNoDocuments {
        response.Failure(c, 404, "Event tidak ditemukan")
        return
    }

    if err != nil {
        response.Failure(c, 500, "Gagal mengambil informasi event")
        return
    }

    response.Success(c, 200, event)
}