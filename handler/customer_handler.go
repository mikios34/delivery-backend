package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authpkg "github.com/mikios34/delivery-backend/auth"
	"github.com/mikios34/delivery-backend/courier"
	customerpkg "github.com/mikios34/delivery-backend/customer"
	orderpkg "github.com/mikios34/delivery-backend/order"
)

// CustomerHandler bundles dependencies for customer-related HTTP handlers.
type CustomerHandler struct {
	service  customerpkg.CustomerService
	orders   orderpkg.Repository
	couriers courier.CourierRepository
}

// NewCustomerHandler constructs a CustomerHandler.
func NewCustomerHandler(svc customerpkg.CustomerService) *CustomerHandler {
	// Backwards-compatible constructor; fields can be set via WithRepos in main.
	return &CustomerHandler{service: svc}
}

// WithRepos allows wiring additional dependencies without breaking existing call sites.
func (h *CustomerHandler) WithRepos(orders orderpkg.Repository, couriers courier.CourierRepository) *CustomerHandler {
	h.orders = orders
	h.couriers = couriers
	return h
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

		// Return a consistent principal response with a token for immediate use
		principal := authpkg.Principal{
			UserID:    createdCustomer.UserID.String(),
			CustomerID: createdCustomer.ID.String(),
			Role:      "customer",
			FirstName: p.FirstName,
			LastName:  p.LastName,
			Phone:     p.Phone,
		}
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "dev-insecure-secret-change-me"
		}
		if token, err := authpkg.SignJWT(secret, &principal, 24*time.Hour); err == nil {
			principal.Token = token
		}
		c.JSON(http.StatusCreated, gin.H{"principal": principal})
	}
}

// ActiveOrder returns the customer's current active order (status not in no_nearby_driver, delivered)
// along with assigned driver details if present.
func (h *CustomerHandler) ActiveOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.orders == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "orders repository not configured"})
			return
		}
		customerIDStr := c.GetString("customer_id")
		if customerIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "customer_id missing in context"})
			return
		}
		customerID, err := uuid.Parse(customerIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		// Fetch the most recently updated active order for this customer
		ord, err := h.orders.GetActiveOrderForCustomer(ctx, customerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if ord == nil {
			c.JSON(http.StatusOK, gin.H{"active": false})
			return
		}

		resp := gin.H{
			"active": true,
			"order":  ord,
		}
		if ord.AssignedCourier != nil && h.couriers != nil {
			// Include driver details only when driver is actively involved with the order
			if ord.Status == "accepted" || ord.Status == "picked_up" || ord.Status == "delivered" || ord.Status == "arrived" || ord.Status == "assigned" {
				if user, err := h.couriers.GetUserByCourierID(ctx, *ord.AssignedCourier); err == nil {
					driver := gin.H{
						"id":    *ord.AssignedCourier,
						"name":  user.FirstName + " " + user.LastName,
						"phone": user.Phone,
					}
					if user.ProfilePicture != nil {
						driver["profile_picture"] = *user.ProfilePicture
					}
					resp["assigned_driver"] = driver
				}
			}
		}
		c.JSON(http.StatusOK, resp)
	}
}
