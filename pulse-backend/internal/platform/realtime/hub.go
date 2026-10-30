// Package realtime pushes events (new readings, alert changes) to the
// dashboards of the user who owns the device.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second  // max time for one write before the client is dropped
	pongWait   = 60 * time.Second  // client must answer a ping within this
	pingPeriod = pongWait * 9 / 10 // how often the server pings
	sendBuffer = 64                // queued messages per client before it counts as stalled
	maxReadMsg = 512               // clients don't send data; this only covers control frames
)

// Envelope is the shape of every message: {"type": "reading", "data": {...}}.
type Envelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type client struct {
	hub    *Hub
	userID int64
	conn   *websocket.Conn
	send   chan []byte
	once   sync.Once
}

// Hub tracks connected clients per user. Publishing never blocks on the
// network: each client has its own buffered queue and writer goroutine, and a
// client whose queue is full is disconnected instead of slowing ingestion.
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[int64]map[*client]struct{})}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if set, ok := h.clients[c.userID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, c.userID)
		}
	}
	h.mu.Unlock()
	c.close()
}

// Publish implements the Notifier interfaces of the readings and alerts features.
func (h *Hub) Publish(userID int64, eventType string, data any) {
	msg, err := json.Marshal(Envelope{Type: eventType, Data: data})
	if err != nil {
		log.Printf("ws: marshal %s event: %v", eventType, err)
		return
	}

	h.mu.RLock()
	var stalled []*client
	for c := range h.clients[userID] {
		select {
		case c.send <- msg:
		default:
			stalled = append(stalled, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range stalled {
		log.Printf("ws: client of user %d is not keeping up, disconnecting", userID)
		h.unregister(c)
	}
}

// CloseAll disconnects every client (used on shutdown).
func (h *Hub) CloseAll() {
	h.mu.Lock()
	all := h.clients
	h.clients = make(map[int64]map[*client]struct{})
	h.mu.Unlock()
	for _, set := range all {
		for c := range set {
			c.close()
		}
	}
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.send)
	})
}

// writePump is the only goroutine that writes to the connection
// (gorilla/websocket allows one concurrent writer).
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.hub.unregister(c)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.hub.unregister(c)
				return
			}
		}
	}
}

// readPump discards client messages but is needed to process pongs and to
// notice disconnects, which surface as read errors.
func (c *client) readPump() {
	defer c.hub.unregister(c)
	c.conn.SetReadLimit(maxReadMsg)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
