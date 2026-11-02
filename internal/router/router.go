package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Ranwiesiel/livesync-api/internal/handlers"
	"github.com/Ranwiesiel/livesync-api/internal/middleware"
)

func New(
	event *handlers.EventHandler,
	storage *handlers.StorageHandler,
	photo *handlers.PhotoHandler,
	ws http.Handler,
	corsOrigins []string,
) *gin.Engine {
	r := gin.New()

	r.Use(
		middleware.CORS(corsOrigins),
		gin.Logger(),
		gin.Recovery(),
	)

	// WebSocket berada di luar prefix /api.
	r.GET("/ws/events/:code", gin.WrapH(ws))

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
