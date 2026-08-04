// internal/ws/hub.go
package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"hydro-backend/internal/domain"
)

// Hub tracks every currently-connected WebSocket client and fans out
// broadcast messages to all of them. This is the standard pattern from
// gorilla/websocket's own examples — a central registry rather than each
// client managing its own state independently, because "send this to
// everyone" needs one place that knows who "everyone" is.
type Hub struct {
	// mu protects the clients map from concurrent access — multiple
	// goroutines (one per client connection, plus whichever goroutine
	// calls Broadcast) all touch this map, so we need a mutex to avoid
	// a race condition (Go's race detector will complain loudly if you
	// forget this).
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Register adds a new client connection to the hub. Called once when a
// client's WebSocket handshake completes.
func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	log.Printf("ws: client connected (total: %d)", len(h.clients))
}

// Unregister removes a client, e.g. when it disconnects or a write to it
// fails. Always call this alongside conn.Close() so the map doesn't
// accumulate dead connections forever (a memory/goroutine leak).
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
		conn.Close()
		log.Printf("ws: client disconnected (total: %d)", len(h.clients))
	}
}

// Broadcast sends a reading to every connected client as JSON. This is
// called from ReadingService right after a successful insert — so both
// HTTP-submitted and MQTT-submitted readings trigger the same broadcast,
// identical to how both already share the same validation and storage path.
func (h *Hub) Broadcast(reading domain.Reading) {
	data, err := json.Marshal(reading)
	if err != nil {
		log.Printf("ws: failed to marshal reading for broadcast: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// A write failure almost always means the client disconnected
			// without a clean close handshake (browser tab closed, network
			// drop). We remove it here rather than leaving it registered
			// and failing forever on every future broadcast.
			log.Printf("ws: write failed, removing client: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
