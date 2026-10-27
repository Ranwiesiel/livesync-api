package models

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestPhotoURLTidakDisimpanDiMongo(t *testing.T) {
	photo := Photo{
		StorageKey: "uploads/2026/10/22/foto.png",
		URL:        "https://signed.test/uploads/foto.png?signature=abc",
	}

	raw, err := bson.Marshal(photo)
	if err != nil {
		t.Fatalf("bson.Marshal() error = %v", err)
	}

	if bytes.Contains(raw, []byte("signed.test")) {
		t.Fatal("URL bertanda tangan tidak boleh ikut tersimpan di Mongo")
	}
	if !bytes.Contains(raw, []byte("uploads/2026/10/22/foto.png")) {
		t.Fatal("storage_key harus tetap tersimpan")
	}
}

func TestPhotoURLTetapAdaDiJSON(t *testing.T) {
	photo := Photo{
		ID:         bson.NewObjectID(),
		StorageKey: "uploads/foto.png",
		URL:        "https://signed.test/uploads/foto.png?signature=abc",
	}

	raw, err := json.Marshal(photo)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if !bytes.Contains(raw, []byte(`"url":"https://signed.test/uploads/foto.png?signature=abc"`)) {
		t.Fatalf("respons JSON harus tetap memuat url, dapat %s", raw)
	}
}
