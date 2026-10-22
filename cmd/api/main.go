
package main

import (
    "context"
    "log"
    "time"

    "github.com/joho/godotenv"
    "github.com/Ranwiesiel/livesync-api/internal/database"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Println("File .env tidak ditemukan")
    }

    client, err := database.GetMongoClient()
    if err != nil {
        log.Fatal("Gagal menginisialisasi MongoDB:", err)
    }

    ctx, cancel := context.WithTimeout(
        context.Background(),
        10*time.Second,
    )
    defer cancel()

    db := client.Database("Livesync")

    log.Println("Database siap:", db.Name())

    _ = ctx
}