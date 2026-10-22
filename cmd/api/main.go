
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/Ranwiesiel/livesync-api/internal/database"
	"github.com/Ranwiesiel/livesync-api/internal/handlers"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, menggunakan environment variable")
	}

	client, err := database.GetMongoClient()
	if err != nil {
		log.Fatal("Gagal menginisialisasi MongoDB:", err)
	}

	db := client.Database("Livesync")
	log.Println("Database siap:", db.Name())

	collection := db.Collection("events")
	eventHandler := handlers.NewEventHandler(collection)

	router := gin.Default()

	api := router.Group("/api")
	{
		api.POST("/events", eventHandler.CreateEvent)
		api.GET("/events/:code", eventHandler.GetEvent)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server berjalan pada port:", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Gagal menjalankan server:", err)
	}
}
