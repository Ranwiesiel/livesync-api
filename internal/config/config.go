package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// StorageConfig menyimpan kredensial yang dibutuhkan untuk Cloudflare R2.
type StorageConfig struct {
	APITokenCF      string
	AccessKeyID     string
	SecretAccessKey string
	S3APIEndpoint   string
	MongoDBURI      string
}

// LoadEnv memuat .env ke environment proses, lalu mengembalikan konfigurasi
func LoadEnv() StorageConfig {
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, memakai environment yang sudah ada")
	}

	return StorageConfig{
		APITokenCF:      os.Getenv("API_TOKEN_CF"),
		AccessKeyID:     os.Getenv("ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("SECRET_ACCESS_KEY"),
		S3APIEndpoint:   os.Getenv("S3_API_ENDPOINT"),
		MongoDBURI:      os.Getenv("MONGODB_URI"),
	}
}
