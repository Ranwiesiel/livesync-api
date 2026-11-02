package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
)

// BroadcastMessage berisi pesan yang akan dikirim ke satu room.
type BroadcastMessage struct {
	EventID string
	Data    []byte
}

type Hub struct {
	// Room berisi kumpulan client berdasarkan Event ID.
	rooms map[string]map[*Client]bool

	// Channel pengelolaan client.
	register   chan *Client
	unregister chan *Client
	broadcast  chan BroadcastMessage

	// Melindungi rooms dari concurrent access.
	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan BroadcastMessage),
	}
}

// Run menjalankan event loop Hub.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()

			if h.rooms[client.eventID] == nil {
				h.rooms[client.eventID] = make(map[*Client]bool)
			}

			h.rooms[client.eventID][client] = true

			h.mu.Unlock()

		case client := <-h.unregister:
			h.removeClient(client)

		case message := <-h.broadcast:
			var slowClients []*Client

			h.mu.RLock()

			for client := range h.rooms[message.EventID] {
				select {
				case client.send <- message.Data:
				default:
					slowClients = append(slowClients, client)
				}
			}

			h.mu.RUnlock()

			// Hapus client setelah read lock dilepas.
			for _, client := range slowClients {
				h.removeClient(client)
			}
		}
	}
}

// removeClient menghapus client dari room.
func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()

	if clients, exists := h.rooms[client.eventID]; exists {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)
		}

		if len(clients) == 0 {
			delete(h.rooms, client.eventID)
		}
	}

	h.mu.Unlock()
}

// Broadcast mengirim pesan ke room berdasarkan Event ID.
func (h *Hub) Broadcast(eventID string, data []byte) {
	h.broadcast <- BroadcastMessage{
		EventID: eventID,
		Data:    data,
	}
}

// BroadcastToRoom mengirim message ke semua client di room eventID.
// Room lain tidak menerima apa pun.
func (h *Hub) BroadcastToRoom(eventID string, message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("websocket: encode pesan: %w", err)
	}

	h.Broadcast(eventID, data)
	return nil
}

// RoomSize mengembalikan jumlah koneksi dalam suatu room.
func (h *Hub) RoomSize(eventID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.rooms[eventID])
}
