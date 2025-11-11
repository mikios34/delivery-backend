package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mikios34/delivery-backend/entity"
	"github.com/mikios34/delivery-backend/realtime"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type WSHandler struct {
	hub               *realtime.Hub
	onCourierLocation func(courierID string, lat, lng *float64)
	orders            interface { // minimal interface to avoid import cycle
		ListActiveOrdersForCustomer(ctx context.Context, customerID uuid.UUID) ([]entity.Order, error)
	}
}

func NewWSHandler(hub *realtime.Hub) *WSHandler { return &WSHandler{hub: hub} }

func (h *WSHandler) WithCourierLocationHandler(fn func(courierID string, lat, lng *float64)) *WSHandler {
	h.onCourierLocation = fn
	return h
}

// WithOrders wires an orders repository for initial sync on customer connect.
func (h *WSHandler) WithOrders(orders interface {
	ListActiveOrdersForCustomer(ctx context.Context, customerID uuid.UUID) ([]entity.Order, error)
}) *WSHandler {
	h.orders = orders
	return h
}

// CourierSocket upgrades to WS and registers the courier connection.
func (h *WSHandler) CourierSocket() gin.HandlerFunc {
	return func(c *gin.Context) {
		// auth + role middleware should run before this handler
		courierID := c.GetString("courier_id")
		if courierID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "courier_id missing in context"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		h.hub.RegisterCourier(courierID, conn)
		// read loop: handle incoming events
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				h.hub.UnregisterCourier(courierID)
				break
			}
			var msg struct {
				Event string          `json:"event"`
				Data  json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg.Event {
			case "location.update":
				var p struct {
					Latitude  *float64 `json:"latitude"`
					Longitude *float64 `json:"longitude"`
				}
				if err := json.Unmarshal(msg.Data, &p); err == nil && h.onCourierLocation != nil {
					h.onCourierLocation(courierID, p.Latitude, p.Longitude)
				}
			default:
				// ignore
			}
		}
	}
}

// CustomerSocket upgrades to WS and registers the customer connection.
func (h *WSHandler) CustomerSocket() gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.GetString("customer_id")
		if customerID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "customer_id missing in context"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		h.hub.RegisterCustomer(customerID, conn)
		// On connect, push current active orders snapshot if repository is available
		if h.orders != nil {
			// lazy imports to avoid tight coupling
			type snapshot struct {
				Orders []entity.Order `json:"orders"`
			}
			if id, err := uuid.Parse(customerID); err == nil {
				// give a short-lived context
				ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
				defer cancel()
				if list, err := h.orders.ListActiveOrdersForCustomer(ctx, id); err == nil {
					_ = h.hub.NotifyCustomer(customerID, "order.sync", snapshot{Orders: list})
				}
			}
		}
		// Currently, no inbound customer events are expected; maintain connection until closed.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				h.hub.UnregisterCustomer(customerID)
				break
			}
		}
	}
}
