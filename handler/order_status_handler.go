package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/courier"
	"github.com/mikios34/delivery-backend/dispatch"
	"github.com/mikios34/delivery-backend/entity"
	orderpkg "github.com/mikios34/delivery-backend/order"
	"github.com/mikios34/delivery-backend/realtime"
)

type OrderStatusHandler struct {
	svc      orderpkg.Service
	couriers courier.CourierRepository
	dispatch dispatch.Service
}

func NewOrderStatusHandler(svc orderpkg.Service, couriers courier.CourierRepository) *OrderStatusHandler {
	return &OrderStatusHandler{svc: svc, couriers: couriers}
}

// WithDispatch wires the dispatch service for reassignment logic (e.g., on decline).
func (h *OrderStatusHandler) WithDispatch(d dispatch.Service) *OrderStatusHandler {
	h.dispatch = d
	return h
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
		// For decline: attempt immediate reassignment and avoid sending a 'declined' notification.
		if target == entity.OrderDeclined && h.dispatch != nil {
			if reassigned, _, err := h.dispatch.ReassignAfterDecline(ctx, oid, cid); err == nil && reassigned != nil {
				// Use the updated state after reassignment (assigned or no_nearby_driver)
				updated = reassigned
			}
			c.JSON(http.StatusOK, updated)
			return
		}

		// Notify customer about status change (single generic event) for non-decline states
		if v, exists := c.Get("hub"); exists {
			if hub, ok := v.(*realtime.Hub); ok && hub != nil {
				payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(updated.Status)}
				// For accepted, picked_up, delivered include courier name + phone + profile picture
				if target == entity.OrderAccepted || target == entity.OrderPickedUp || target == entity.OrderDelivered {
					if cour, err := h.couriers.GetCourierByID(ctx, cid); err == nil {
						if user, err := h.couriers.GetUserByID(ctx, cour.UserID); err == nil {
							name := strings.TrimSpace(user.FirstName + " " + user.LastName)
							phone := user.Phone
							payload.CourierName = &name
							payload.CourierPhone = &phone
							if user.ProfilePicture != nil {
								payload.CourierProfilePicture = user.ProfilePicture
							}
						}
					}
				}
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

// CancelCustomer allows a customer to cancel an order.
// Payload: {"order_id": "uuid"}
func (h *OrderStatusHandler) CancelCustomer() gin.HandlerFunc {
	type payload struct {
		OrderID string `json:"order_id" binding:"required"`
	}
	return func(c *gin.Context) {
		var p payload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		oid, err := uuid.Parse(p.OrderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		updated, err := h.svc.CancelByCustomer(ctx, oid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if v, exists := c.Get("hub"); exists {
			if hub, ok := v.(*realtime.Hub); ok && hub != nil {
				payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(updated.Status)}
				_ = hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
				// Also notify assigned courier if still present (before clear assignment happened in service)
				if updated.AssignedCourier != nil {
					_ = hub.Notify(updated.AssignedCourier.String(), "order.status", payload)
				}
			}
		}
		c.JSON(http.StatusOK, updated)
	}
}

// CancelCourier allows the assigned courier to cancel an order.
// Payload: {"order_id": "uuid", "courier_id": "uuid"}
func (h *OrderStatusHandler) CancelCourier() gin.HandlerFunc {
	type payload struct {
		OrderID   string `json:"order_id" binding:"required"`
		CourierID string `json:"courier_id" binding:"required"`
	}
	return func(c *gin.Context) {
		var p payload
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

		// 1. Validate permission and state
		ord, err := h.svc.GetOrder(ctx, oid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if ord.AssignedCourier == nil || *ord.AssignedCourier != cid {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: not assigned courier"})
			return
		}
		if ord.Status == entity.OrderPickedUp || ord.Status == entity.OrderDelivered {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot cancel order after pickup or delivery"})
			return
		}
		if ord.Status == entity.OrderCanceledByCustomer || ord.Status == entity.OrderCanceledByCourier {
			c.JSON(http.StatusOK, ord)
			return
		}

		// 2. Reassign (treat as decline/unassign)
		if h.dispatch != nil {
			updated, _, err := h.dispatch.ReassignAfterDecline(ctx, oid, cid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reassign: " + err.Error()})
				return
			}
			c.JSON(http.StatusOK, updated)
			return
		}

		// Fallback if dispatch is not wired
		updated, err := h.svc.CancelByCourier(ctx, oid, cid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if v, exists := c.Get("hub"); exists {
			if hub, ok := v.(*realtime.Hub); ok && hub != nil {
				payload := realtime.OrderStatusPayload{OrderID: updated.ID.String(), Status: string(updated.Status)}
				_ = hub.NotifyCustomer(updated.CustomerID.String(), "order.status", payload)
				// Also notify assigned courier (the canceling courier) so all connected devices stay in sync.
				if updated.AssignedCourier != nil {
					_ = hub.Notify(updated.AssignedCourier.String(), "order.status", payload)
				}
			}
		}
		c.JSON(http.StatusOK, updated)
	}
}
