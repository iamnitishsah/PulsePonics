package realtime

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"pulse-backend/internal/platform/apperr"
	"pulse-backend/internal/platform/httpx"
)

// TokenParser validates an access token and returns the user ID.
type TokenParser interface {
	Parse(token string) (int64, error)
}

// Handler upgrades GET /api/ws?token=<access token>. Browsers cannot set an
// Authorization header on WebSocket requests, so the access token travels in
// the query string; it is short-lived and only checked at connect time.
func Handler(hub *Hub, tokens TokenParser, origins *httpx.OriginPolicy) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Same allowlist as CORS. Non-browser clients send no Origin header.
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || origins.Allowed(origin)
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := tokens.Parse(r.URL.Query().Get("token"))
		if err != nil {
			httpx.WriteError(w, r, apperr.Unauthorized("invalid or expired access token"))
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws: upgrade failed: %v", err) // Upgrade already wrote the HTTP error
			return
		}

		c := &client{hub: hub, userID: userID, conn: conn, send: make(chan []byte, sendBuffer)}
		hub.register(c)
		go c.writePump()
		go c.readPump()
	}
}
