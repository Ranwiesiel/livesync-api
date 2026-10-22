package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config menampung seluruh nilai konfigurasi yang dibaca dari environment.
type Config struct {
	Port string

	MongoDBURI    string
	MongoDatabase string

	APITokenCF      string
	AccessKeyID     string
	SecretAccessKey string
	S3APIEndpoint   string
	Bucket          string
}

// LoadEnv memuat .env ke environment proses, lalu mengembalikan Config.
// File .env yang tidak ada bukan error fatal: nilai dari environment asli
// tetap dipakai.
func LoadEnv() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, memakai environment yang sudah ada")
	}

	return Config{
		Port:            envOr("PORT", "8090"),
		MongoDBURI:      os.Getenv("MONGODB_URI"),
		MongoDatabase:   envOr("MONGODB_DATABASE", "Livesync"),
		APITokenCF:      os.Getenv("API_TOKEN_CF"),
		AccessKeyID:     os.Getenv("ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("SECRET_ACCESS_KEY"),
		S3APIEndpoint:   os.Getenv("S3_API_ENDPOINT"),
		Bucket:          envOr("R2_BUCKET", "livesync-dev"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
