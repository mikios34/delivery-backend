package realtime

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu        sync.RWMutex
	byCourier map[string]*websocket.Conn
}

func NewHub() *Hub { return &Hub{byCourier: make(map[string]*websocket.Conn)} }

func (h *Hub) RegisterCourier(courierID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.byCourier[courierID]; ok {
		old.Close()
	}
	h.byCourier[courierID] = conn
}

func (h *Hub) UnregisterCourier(courierID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn, ok := h.byCourier[courierID]; ok {
		conn.Close()
		delete(h.byCourier, courierID)
	}
}

// Notify sends a typed event payload to the courier if connected.
func (h *Hub) Notify(courierID string, event string, payload any) error {
	h.mu.RLock()
	conn, ok := h.byCourier[courierID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	msg := map[string]any{"event": event, "data": payload}
	return conn.WriteJSON(msg)
}

// Helper payloads
type AssignmentPayload struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
}

func Marshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
