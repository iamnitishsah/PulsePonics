package realtime

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// upgrader converts a normal HTTP GET request into a long-lived WebSocket
// connection. This "upgrade" is literally part of the HTTP spec (the
// Upgrade header) — a WebSocket connection starts life as an HTTP request,
// then switches protocols on the same TCP connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// CheckOrigin controls which sites are allowed to open a WebSocket to
	// this server. Returning true unconditionally is fine for local
	// dev/your React frontend during development, but you'd tighten this
	// (check r.Header.Get("Origin") against an allowlist) before any real
	// deployment — otherwise any website could open a connection to your
	// server from a user's browser.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler returns an http.HandlerFunc that upgrades incoming requests to
// WebSocket connections and registers them with the given Hub. Returning a
// closure like this (rather than a method needing the hub some other way)
// is a common Go idiom for handlers that need a dependency injected.
func Handler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws: upgrade failed: %v", err)
			return
		}

		hub.Register(conn)

		// We don't expect the client to send us anything meaningful right
		// now (Phase 3 is push-only: server → client). But we still need
		// a read loop — without one, we'd never notice when the client
		// disconnects, since TCP disconnects surface as read errors, not
		// some separate "onClose" callback like you might expect from
		// higher-level libraries.
		go func() {
			defer hub.Unregister(conn)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					// Any read error (client closed tab, network drop,
					// clean close frame) lands here — we just clean up.
					return
				}
			}
		}()
	}
}
