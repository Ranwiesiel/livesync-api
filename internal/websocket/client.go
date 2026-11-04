package websocket

import (
	"encoding/json"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Ranwiesiel/livesync-api/internal/broadcast"
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

	// Batas ukuran emoji untuk mencegah payload berlebihan.
	maxEmojiSize = 32
)

type Client struct {
	conn    *gorilla.Conn
	send    chan []byte
	eventID string
	hub     *Hub
	ratelimiter *TokenBucket
}

// IncomingMessage adalah struktur pesan yang diterima dari client.
type IncomingMessage struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

// ReactionMessage adalah payload yang dibroadcast ke room.
type ReactionMessage struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji"`
}

// NewClient membuat client baru.
func NewClient(hub *Hub, conn *gorilla.Conn, eventID string) *Client {
	return &Client{
		conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		eventID: eventID,
		hub:     hub,
		ratelimiter: NewTokenBucket(),
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

		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
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

		// Protokol aplikasi menggunakan JSON melalui pesan teks.
		if messageType != gorilla.TextMessage {
			continue
		}

		c.handleMessage(message)
	}
}

// handleMessage memproses pesan yang dikirim client.
func (c *Client) handleMessage(data []byte) {
	log.Printf("[WS] Pesan diterima dari event %s: %s",
        c.eventID,
        string(data),
    )
	var message IncomingMessage

	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf(
			"websocket: invalid JSON event=%s: %v",
			c.eventID,
			err,
		)
		return
	}

	switch message.Type {
	case "SEND_REACTION":
		c.handleReaction(message)

	default:
		log.Printf(
			"websocket: unsupported message type=%s event=%s",
			message.Type,
			c.eventID,
		)
	}
}

// handleReaction memvalidasi dan membroadcast reaksi.
func (c *Client) handleReaction(message IncomingMessage) {
    emoji := strings.TrimSpace(message.Emoji)

    if emoji == "" {
        return
    }

    if !utf8.ValidString(emoji) || len(emoji) > maxEmojiSize {
        log.Printf("websocket: invalid reaction from event %s", c.eventID)
        return
    }

	if !c.ratelimiter.Allow() {
		log.Printf("websocket: reaction rate limit exceeded for event %s", c.eventID)
		return
	}

    published := broadcast.Publish(broadcast.Event{
        Type:    broadcast.TypeLiveReaction,
        EventID: c.eventID,
        Payload: map[string]string{
            "emoji": emoji,
        },
    })

    if !published {
        log.Printf("websocket: reaction channel full for event %s", c.eventID)
    } else {
		log.Printf(
			"[REACTION] Berhasil publish: event=%s emoji=%s",
			c.eventID,
			emoji,
		)
	}
}

// writePump mengirim pesan dan heartbeat ke client.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Channel ditutup oleh Hub.
				c.conn.WriteMessage(
					gorilla.CloseMessage,
					[]byte{},
				)
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.conn.WriteMessage(
				gorilla.TextMessage,
				message,
			); err != nil {
				log.Printf(
					"websocket: write error event=%s: %v",
					c.eventID,
					err,
				)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.conn.WriteMessage(
				gorilla.PingMessage,
				nil,
			); err != nil {
				log.Printf(
					"websocket: ping error event=%s: %v",
					c.eventID,
					err,
				)
				return
			}
		}
	}
}
