package websocket

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	gorilla "github.com/gorilla/websocket"
)

var eventCodePattern = regexp.MustCompile(`^\d{6}$`)

// EventValidator mengembalikan ID event (bukan kode) untuk dijadikan
// key room. String kosong berarti event tidak ditemukan/tidak aktif.
type EventValidator func(r *http.Request, code string) (string, error)

type Handler struct {
	hub       *Hub
	validator EventValidator
	upgrader  gorilla.Upgrader
}

func NewHandler(
	hub *Hub,
	validator EventValidator,
	allowedOrigins []string,
) *Handler {
	return &Handler{
		hub:       hub,
		validator: validator,
		upgrader: gorilla.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")

				// Jika Origin tidak ada, tolak secara default.
				if origin == "" {
					return false
				}

				return isAllowedOrigin(origin, allowedOrigins)
			},
		},
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	requestOrigin, err := url.Parse(origin)
	if err != nil ||
		requestOrigin.Scheme == "" ||
		requestOrigin.Host == "" ||
		requestOrigin.User != nil {
		return false
	}

	for _, allowed := range allowedOrigins {
		allowedOrigin, err := url.Parse(strings.TrimSpace(allowed))
		if err != nil {
			continue
		}

		if strings.EqualFold(requestOrigin.Scheme, allowedOrigin.Scheme) &&
			strings.EqualFold(requestOrigin.Host, allowedOrigin.Host) &&
			allowedOrigin.User == nil &&
			allowedOrigin.Path == "" &&
			allowedOrigin.RawQuery == "" &&
			allowedOrigin.Fragment == "" {
			return true
		}
	}

	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) != 3 ||
		parts[0] != "ws" ||
		parts[1] != "events" {
		http.NotFound(w, r)
		return
	}

	code := parts[2]

	if !eventCodePattern.MatchString(code) {
		http.Error(w, "Kode event tidak valid", http.StatusBadRequest)
		return
	}

	if h.validator == nil {
		http.Error(w, "Event validator belum dikonfigurasi",
			http.StatusInternalServerError)
		return
	}

	eventID, err := h.validator(r, code)
	if err != nil {
		http.Error(w, "Gagal memvalidasi event",
			http.StatusInternalServerError)
		return
	}

	if eventID == "" {
		http.Error(w, "Event tidak ditemukan atau tidak aktif",
			http.StatusNotFound)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(h.hub, conn, eventID)

	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}
