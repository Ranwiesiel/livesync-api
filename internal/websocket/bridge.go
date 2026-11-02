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

			data, err := broadcast.Encode(event)
			if err != nil {
				log.Printf("websocket: encode event %s: %v", event.Type, err)
				continue
			}

			hub.Broadcast(event.EventID, data)
		}
	}
}
