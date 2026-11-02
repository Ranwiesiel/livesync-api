package websocket

import (
	"context"
	"log"

	"github.com/Ranwiesiel/livesync-api/internal/broadcast"
)

// RunBridge meneruskan event dari channel broadcast ke Hub. Worker tidak
// perlu tahu WebSocket; cukup Publish ke channel.
func RunBridge(ctx context.Context, hub *Hub, events <-chan broadcast.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			// Hanya room event terkait yang menerima pesan ini.
			err := hub.BroadcastToRoom(event.EventID, broadcast.Message{
				Type:    event.Type,
				EventID: event.EventID,
				Payload: event.Payload,
			})
			if err != nil {
				log.Printf("websocket: broadcast %s: %v", event.Type, err)
			}
		}
	}
}
