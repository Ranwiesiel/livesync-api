package broadcast

import (
	"encoding/json"
	"time"
)

// Tipe pesan yang dipakai bersama API, worker, dan client WebSocket.
const (
	TypeNewPhoto     = "NEW_PHOTO"
	TypeLiveReaction = "LIVE_REACTION"
)

// Event adalah pesan internal sebelum diserialisasi ke client.
type Event struct {
	Type    string
	EventID string
	Payload any
}

// Message adalah bentuk JSON yang dikirim ke client:
// {"type":"NEW_PHOTO","event_id":"...","payload":{...}}
type Message struct {
	Type    string `json:"type"`
	EventID string `json:"event_id"`
	Payload any    `json:"payload"`
}

// NewPhoto adalah payload untuk tipe NEW_PHOTO.
type NewPhoto struct {
	ID           string    `json:"id"`
	EventID      string    `json:"event_id"`
	StorageKey   string    `json:"storage_key"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	BlurHash     string    `json:"blur_hash"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Uploader     string    `json:"uploader"`
	CreatedAt    time.Time `json:"created_at"`
}

type NewPhotoPayload struct {
	Photo NewPhoto `json:"photo"`
}

// Encode mengubah event internal menjadi payload WebSocket.
func Encode(event Event) ([]byte, error) {
	return json.Marshal(Message{
		Type:    event.Type,
		EventID: event.EventID,
		Payload: event.Payload,
	})
}

// Events adalah channel perantara. Worker cukup menulis ke sini tanpa tahu
// WebSocket; bridge di main yang meneruskannya ke Hub.
var Events = make(chan Event, 256)

// Publish mengirim event tanpa memblokir worker. false berarti channel penuh;
// update live bersifat best-effort karena database tetap sumber kebenaran.
func Publish(event Event) bool {
	select {
	case Events <- event:
		return true
	default:
		return false
	}
}
