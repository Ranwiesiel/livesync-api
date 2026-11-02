package websocket

import (
	"log"
	"time"

	gorilla "github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second

	// Server mengirim ping setiap 30 detik.
	pingPeriod = 30 * time.Second

	// Koneksi dianggap mati jika tidak menerima pong selama 60 detik.
	pongWait = 60 * time.Second

	maxMessageSize = 64 * 1024
	sendBufferSize = 256
)

type Client struct {
	conn    *gorilla.Conn
	send    chan []byte
	eventID string
	hub     *Hub
}

// NewClient membuat client baru.
func NewClient(hub *Hub, conn *gorilla.Conn, eventID string) *Client {
	return &Client{
		conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		eventID: eventID,
		hub:     hub,
	}
}

// readPump membaca pesan dari koneksi WebSocket.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	// Koneksi harus menerima pong sebelum deadline.
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(appData string) error {
		log.Printf("Pong received from event %s", c.eventID)

		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if gorilla.IsUnexpectedCloseError(
				err,
				gorilla.CloseGoingAway,
				gorilla.CloseAbnormalClosure,
			) {
				log.Printf(
					"websocket: read error event=%s: %v",
					c.eventID,
					err,
				)
			}
			break
		}

		// Pada tahap ini client hanya diperbolehkan
		// mengirim pesan teks atau binary.
		// Belum ada perintah client yang diproses.
		if messageType != gorilla.TextMessage &&
			messageType != gorilla.BinaryMessage {
			continue
		}

		// Pesan dari client belum dibroadcast.
		// Nanti dapat diintegrasikan dengan protokol event LiveSync.
		_ = message
	}
}

// writePump mengirim pesan dan heartbeat ke client.
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}

			if err := c.conn.WriteMessage(gorilla.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.WriteMessage(gorilla.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
