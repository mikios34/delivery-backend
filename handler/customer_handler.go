package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	customerpkg "github.com/mikios34/delivery-backend/customer"
)

// CustomerHandler bundles dependencies for customer-related HTTP handlers.
type CustomerHandler struct {
	service customerpkg.CustomerService
}

// NewCustomerHandler constructs a CustomerHandler.
func NewCustomerHandler(svc customerpkg.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: svc}
}

type registerCustomerPayload struct {
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	FirebaseUID string `json:"firebase_uid" binding:"required"`
}

// RegisterCustomer registers a customer (creates user and customer profile).
func (h *CustomerHandler) RegisterCustomer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p registerCustomerPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}

		req := customerpkg.RegisterCustomerRequest{
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Phone:       p.Phone,
			FirebaseUID: p.FirebaseUID,
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		createdCustomer, err := h.service.RegisterCustomer(ctx, req)
		if err != nil {
			switch err.Error() {
			case "customer with this phone already exists":
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register customer", "detail": err.Error()})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "customer created; phone verified (frontend)",
			"customer": gin.H{
				"id":      createdCustomer.ID,
				"user_id": createdCustomer.UserID,
			},
		})
	}
}
