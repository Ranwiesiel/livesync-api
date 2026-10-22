package config

import (
	"log"
	"os"
	"strings"

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

	// CORSOrigins adalah daftar origin browser yang boleh memanggil API.
	CORSOrigins []string
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
		CORSOrigins: splitList(envOr("CORS_ALLOWED_ORIGINS",
			"http://localhost:5173,http://127.0.0.1:5173")),
	}
}

// splitList memecah daftar dipisah koma dan membuang entri kosong.
func splitList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
