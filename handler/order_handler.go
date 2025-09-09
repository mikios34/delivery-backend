package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dispatchsvc "github.com/mikios34/delivery-backend/dispatch"
	orderpkg "github.com/mikios34/delivery-backend/order"
)

type OrderHandler struct {
	service  orderpkg.Service
	dispatch dispatchsvc.Service
}

func NewOrderHandler(svc orderpkg.Service, d dispatchsvc.Service) *OrderHandler {
	return &OrderHandler{service: svc, dispatch: d}
}

type createOrderPayload struct {
	CustomerID     string   `json:"customer_id" binding:"required"`
	TypeID         string   `json:"type_id" binding:"required"`
	ReceiverPhone  string   `json:"receiver_phone" binding:"required"`
	PickupAddress  string   `json:"pickup_address" binding:"required"`
	PickupLat      *float64 `json:"pickup_lat"`
	PickupLng      *float64 `json:"pickup_lng"`
	DropoffAddress string   `json:"dropoff_address" binding:"required"`
	DropoffLat     *float64 `json:"dropoff_lat"`
	DropoffLng     *float64 `json:"dropoff_lng"`
}

func (h *OrderHandler) CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p createOrderPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		cid, err := uuid.Parse(p.CustomerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
		tid, err := uuid.Parse(p.TypeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type_id"})
			return
		}
		req := orderpkg.CreateOrderRequest{
			CustomerID:     cid,
			TypeID:         tid,
			ReceiverPhone:  p.ReceiverPhone,
			PickupAddress:  p.PickupAddress,
			PickupLat:      p.PickupLat,
			PickupLng:      p.PickupLng,
			DropoffAddress: p.DropoffAddress,
			DropoffLat:     p.DropoffLat,
			DropoffLng:     p.DropoffLng,
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		created, err := h.service.CreateOrder(ctx, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order", "detail": err.Error()})
			return
		}
		// auto-dispatch synchronously for now
		assignedOrder, assignedCourier, derr := h.dispatch.FindAndAssign(ctx, created.ID)
		if derr != nil {
			// return created order without assignment but include error info
			c.JSON(http.StatusCreated, gin.H{"order": created, "dispatch_error": derr.Error()})
			return
		}
		if assignedCourier == nil {
			c.JSON(http.StatusCreated, gin.H{"order": assignedOrder, "message": "no available couriers"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"order": assignedOrder, "assigned_courier_id": assignedCourier.ID})
	}
}

func (h *OrderHandler) ListOrderTypes() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		types, err := h.service.ListOrderTypes(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, types)
	}
}
