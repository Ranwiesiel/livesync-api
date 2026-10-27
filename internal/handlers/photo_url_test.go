package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ranwiesiel/livesync-api/internal/models"
)

func TestValidateStorageKey(t *testing.T) {
	valid := []string{
		"uploads/foto.png",
		"uploads/2026/10/22/9f3a-foto.jpg",
	}
	for _, key := range valid {
		if err := validateStorageKey(key); err != nil {
			t.Errorf("validateStorageKey(%q) error = %v, ingin nil", key, err)
		}
	}

	invalid := []string{
		"",
		"foto.png",
		"uploads/",
		"uploads/../foto.png",
		"uploads//foto.png",
		"uploads/foto.png\x00.jpg",
	}
	for _, key := range invalid {
		if err := validateStorageKey(key); err == nil {
			t.Errorf("validateStorageKey(%q) = nil, ingin error", key)
		}
	}
}

type fakeSigner struct {
	keys []string
	err  error
}

func (f *fakeSigner) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	f.keys = append(f.keys, key)

	if f.err != nil {
		return "", f.err
	}

	return "https://signed.test/" + key, nil
}

func TestSignPhotoURLs(t *testing.T) {
	photos := []models.Photo{
		{StorageKey: "uploads/a.png", URL: "https://lama.test/a.png"},
		{StorageKey: "uploads/b.png", URL: "https://lama.test/b.png"},
	}
	signer := &fakeSigner{}

	if err := signPhotoURLs(context.Background(), signer, time.Hour, photos); err != nil {
		t.Fatalf("signPhotoURLs() error = %v", err)
	}

	if len(signer.keys) != 2 ||
		signer.keys[0] != "uploads/a.png" ||
		signer.keys[1] != "uploads/b.png" {
		t.Fatalf("key yang ditandatangani = %v", signer.keys)
	}

	if photos[0].URL != "https://signed.test/uploads/a.png" {
		t.Errorf("URL foto pertama = %q", photos[0].URL)
	}
	if photos[1].URL != "https://signed.test/uploads/b.png" {
		t.Errorf("URL foto kedua = %q", photos[1].URL)
	}
}

func TestSignPhotoURLsError(t *testing.T) {
	signer := &fakeSigner{err: errors.New("signing gagal")}
	photos := []models.Photo{{StorageKey: "uploads/a.png", URL: "https://lama.test/a.png"}}

	err := signPhotoURLs(context.Background(), signer, time.Hour, photos)
	if err == nil {
		t.Fatal("signPhotoURLs() error = nil, ingin error")
	}

	if photos[0].URL != "https://lama.test/a.png" {
		t.Errorf("URL tidak boleh berubah saat signing gagal, dapat %q", photos[0].URL)
	}
}
