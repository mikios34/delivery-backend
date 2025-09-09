package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mikios34/delivery-backend/realtime"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

type WSHandler struct{ hub *realtime.Hub }

func NewWSHandler(hub *realtime.Hub) *WSHandler { return &WSHandler{hub: hub} }

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
		// keep connection open; read loop to detect close
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				h.hub.UnregisterCourier(courierID)
				break
			}
		}
	}
}
