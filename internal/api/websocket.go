package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/saikrishnans/job-scheduler/internal/metrics"
	"github.com/saikrishnans/job-scheduler/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in dev; tighten in prod via env.
		return true
	},
}

// client represents a single WebSocket connection.
type client struct {
	conn   *websocket.Conn
	sendCh chan []byte
}

// Hub manages all active WebSocket clients and broadcasts.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	c := &client{conn: conn, sendCh: make(chan []byte, 256)}
	h.register(c)
	metrics.WebSocketConnections.Inc()

	go c.writePump(func() {
		h.unregister(c)
		metrics.WebSocketConnections.Dec()
	})
	go c.readPump(func() { h.unregister(c) })
}

// Broadcast serialises msg and sends it to every connected client.
func (h *Hub) Broadcast(msg models.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws marshal: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.sendCh <- data:
		default:
			// Slow client — drop the message rather than blocking.
		}
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.sendCh)
	}
	h.mu.Unlock()
}

// writePump pumps messages from sendCh to the WebSocket connection.
func (c *client) writePump(onClose func()) {
	defer func() {
		c.conn.Close()
		onClose()
	}()

	for data := range c.sendCh {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws write: %v", err)
			return
		}
	}
}

// readPump drains incoming frames (ping/pong, close) and detects disconnects.
func (c *client) readPump(onClose func()) {
	defer func() {
		c.conn.Close()
		onClose()
	}()

	c.conn.SetReadLimit(512)
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws read: %v", err)
			}
			return
		}
	}
}
