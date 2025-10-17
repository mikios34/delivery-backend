package realtime

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu         sync.RWMutex
	byCourier  map[string]*websocket.Conn
	byCustomer map[string]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{byCourier: make(map[string]*websocket.Conn), byCustomer: make(map[string]*websocket.Conn)}
}

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

// Customer WebSocket management
func (h *Hub) RegisterCustomer(customerID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.byCustomer[customerID]; ok {
		old.Close()
	}
	h.byCustomer[customerID] = conn
}

func (h *Hub) UnregisterCustomer(customerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn, ok := h.byCustomer[customerID]; ok {
		conn.Close()
		delete(h.byCustomer, customerID)
	}
}

// NotifyCustomer sends an event to the customer if connected.
func (h *Hub) NotifyCustomer(customerID string, event string, payload any) error {
	h.mu.RLock()
	conn, ok := h.byCustomer[customerID]
	h.mu.RUnlock()
	if !ok {
		log.Printf("ws: customer %s not connected; drop event %s", customerID, event)
		return nil
	}
	msg := map[string]any{"event": event, "data": payload}
	if err := conn.WriteJSON(msg); err != nil {
		log.Printf("ws: write to customer %s failed for event %s: %v", customerID, event, err)
		return err
	}
	return nil
}

// Helper payloads
type AssignmentPayload struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
}

// OrderStatusPayload is sent to customers on status changes.
type OrderStatusPayload struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

func Marshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
