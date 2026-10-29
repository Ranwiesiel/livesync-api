package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Ranwiesiel/livesync-api/internal/handlers"
	"github.com/Ranwiesiel/livesync-api/internal/middleware"
)

// New merakit engine Gin beserta seluruh route aplikasi.
//
// Ini satu-satunya tempat daftar endpoint terlihat utuh. Handler tidak perlu
// tahu prefix URL-nya, dan main tidak perlu tahu route apa pun.
func New(event *handlers.EventHandler, storage *handlers.StorageHandler, photo *handlers.PhotoHandler, corsOrigins []string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(corsOrigins), gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	{
		api.POST("/events", event.CreateEvent)
		api.GET("/events/:code", event.GetEvent)

		api.POST("/media/confirm", photo.ConfirmPhoto)
		api.POST("/media/confirm-batch", photo.ConfirmBatch)
		api.GET("/media/events/:id/photos", photo.GetEventPhotos)

		media := api.Group("/media")
		media.POST("/presign-upload", storage.PresignUpload)
	}

	return r
}
