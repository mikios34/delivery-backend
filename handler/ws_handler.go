package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mikios34/delivery-backend/realtime"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type WSHandler struct {
	hub               *realtime.Hub
	onCourierLocation func(courierID string, lat, lng *float64)
}

func NewWSHandler(hub *realtime.Hub) *WSHandler { return &WSHandler{hub: hub} }

func (h *WSHandler) WithCourierLocationHandler(fn func(courierID string, lat, lng *float64)) *WSHandler {
	h.onCourierLocation = fn
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
