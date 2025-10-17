package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/entity"
	orderpkg "github.com/mikios34/delivery-backend/order"
	"github.com/mikios34/delivery-backend/realtime"
)

type OrderStatusHandler struct{ svc orderpkg.Service }

func NewOrderStatusHandler(svc orderpkg.Service) *OrderStatusHandler {
	return &OrderStatusHandler{svc: svc}
}

type statusPayload struct {
	OrderID   string `json:"order_id" binding:"required"`
	CourierID string `json:"courier_id" binding:"required"`
}

func (h *OrderStatusHandler) update(target entity.OrderStatus) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p statusPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		oid, err := uuid.Parse(p.OrderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order_id"})
			return
		}
		cid, err := uuid.Parse(p.CourierID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid courier_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		updated, err := h.svc.UpdateStatus(ctx, oid, target, &cid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Notify customer about status change (single generic event)
		if v, exists := c.Get("hub"); exists {
			if hub, ok := v.(*realtime.Hub); ok && hub != nil {
				payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(updated.Status)}
				_ = hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
			}
		}
		c.JSON(http.StatusOK, updated)
	}
}

func (h *OrderStatusHandler) Accept() gin.HandlerFunc    { return h.update(entity.OrderAccepted) }
func (h *OrderStatusHandler) Decline() gin.HandlerFunc   { return h.update(entity.OrderDeclined) }
func (h *OrderStatusHandler) Arrived() gin.HandlerFunc   { return h.update(entity.OrderArrived) }
func (h *OrderStatusHandler) Picked() gin.HandlerFunc    { return h.update(entity.OrderPickedUp) }
func (h *OrderStatusHandler) Delivered() gin.HandlerFunc { return h.update(entity.OrderDelivered) }
