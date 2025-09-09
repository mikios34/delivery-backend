package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mikios34/delivery-backend/dispatch"
)

type DispatchHandler struct{ svc dispatch.Service }

func NewDispatchHandler(s dispatch.Service) *DispatchHandler { return &DispatchHandler{svc: s} }

type dispatchPayload struct {
	OrderID string `json:"order_id" binding:"required"`
}

func (h *DispatchHandler) Dispatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p dispatchPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		id, err := uuid.Parse(p.OrderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		ord, courier, err := h.svc.Dispatch(ctx, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"order": ord, "courier": courier})
	}
}
