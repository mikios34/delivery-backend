package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
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
	FirstName      string  `json:"first_name" binding:"required"`
	LastName       string  `json:"last_name" binding:"required"`
	Phone          string  `json:"phone" binding:"required"`
	FirebaseUID    string  `json:"firebase_uid" binding:"required"`
	ProfilePicture *string `json:"profile_picture,omitempty"`
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
			FirstName:      p.FirstName,
			LastName:       p.LastName,
			Phone:          p.Phone,
			FirebaseUID:    p.FirebaseUID,
			ProfilePicture: p.ProfilePicture,
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
			UserID:     createdCustomer.UserID.String(),
			CustomerID: createdCustomer.ID.String(),
			Role:       "customer",
			FirstName:  p.FirstName,
			LastName:   p.LastName,
			Phone:      p.Phone,
		}
		// If repository has GetUserByID, fetch to include persisted profile picture
		if h.service != nil {
			// best-effort enrich: safe cast to access repo via interface
		}
		// Enrich via orders/couriers not applicable here; try reading user if available through customer repo
		// Since handler doesn't have direct repo, rely on payload fallback and JWT claims
		if principal.ProfilePicture == nil && p.ProfilePicture != nil {
			principal.ProfilePicture = p.ProfilePicture
		}
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "dev-insecure-secret-change-me"
		}
		if token, err := authpkg.SignJWT(secret, &principal, 15*time.Minute, "access"); err == nil {
			principal.Token = token
		}
		if refresh, err := authpkg.SignJWT(secret, &principal, 30*24*time.Hour, "refresh"); err == nil {
			principal.RefreshToken = refresh
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
			// Include driver details only after acceptance: accepted, arrived, picked_up, delivered
			if ord.Status == "accepted" || ord.Status == "picked_up" || ord.Status == "delivered" || ord.Status == "arrived" {
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

// ActiveOrders returns all active orders for the authenticated customer.
func (h *CustomerHandler) ActiveOrders() gin.HandlerFunc {
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

		list, err := h.orders.ListActiveOrdersForCustomer(ctx, customerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Enrich each order with assigned_driver when status is post-acceptance
		enriched := make([]gin.H, 0, len(list))
		for i := range list {
			o := list[i]
			m := gin.H{
				"id":                    o.ID,
				"customer_id":           o.CustomerID,
				"assigned_courier":      o.AssignedCourier,
				"type_id":               o.TypeID,
				"vehicle_type_id":       o.VehicleTypeID,
				"receiver_phone":        o.ReceiverPhone,
				"pickup_address":        o.PickupAddress,
				"pickup_lat":            o.PickupLat,
				"pickup_lng":            o.PickupLng,
				"dropoff_address":       o.DropoffAddress,
				"dropoff_lat":           o.DropoffLat,
				"dropoff_lng":           o.DropoffLng,
				"estimated_price_cents": o.EstimatedPriceCents,
				"status":                o.Status,
				"created_at":            o.CreatedAt,
				"updated_at":            o.UpdatedAt,
			}
			if o.AssignedCourier != nil && h.couriers != nil {
				if o.Status == "accepted" || o.Status == "arrived" || o.Status == "picked_up" || o.Status == "delivered" {
					if user, err := h.couriers.GetUserByCourierID(ctx, *o.AssignedCourier); err == nil {
						driver := gin.H{
							"id":    *o.AssignedCourier,
							"name":  user.FirstName + " " + user.LastName,
							"phone": user.Phone,
						}
						if user.ProfilePicture != nil {
							driver["profile_picture"] = *user.ProfilePicture
						}
						m["assigned_driver"] = driver
					}
				}
			}
			enriched = append(enriched, m)
		}
		c.JSON(http.StatusOK, gin.H{"active_orders": enriched})
	}
}

// OrderHistory returns the customer's order history (all statuses), newest first, with pagination.
func (h *CustomerHandler) OrderHistory() gin.HandlerFunc {
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

		// parse pagination
		const (
			defaultLimit = 25
			maxLimit     = 100
		)
		limit := defaultLimit
		offset := 0
		page := 1
		if lStr := c.Query("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		if pStr := c.Query("page"); pStr != "" {
			if p, err := strconv.Atoi(pStr); err == nil && p >= 1 {
				page = p
			}
			offset = (page - 1) * limit
		} else if oStr := c.Query("offset"); oStr != "" {
			if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
				offset = o
				page = (offset / limit) + 1
			}
		}
		if limit > maxLimit {
			limit = maxLimit
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		total, err := h.orders.CountOrdersForCustomer(ctx, customerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count orders", "detail": err.Error()})
			return
		}
		list, err := h.orders.ListOrdersForCustomer(ctx, customerID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch orders", "detail": err.Error()})
			return
		}

		totalPages := 0
		if limit > 0 {
			totalPages = int((total + int64(limit) - 1) / int64(limit))
		}
		hasMore := int64(offset+len(list)) < total

		c.JSON(http.StatusOK, gin.H{
			"count":       total,
			"orders":      list,
			"limit":       limit,
			"offset":      offset,
			"page":        page,
			"total_pages": totalPages,
			"has_more":    hasMore,
		})
	}
}

// CompletedOrders returns delivered orders for the authenticated customer with pagination.
func (h *CustomerHandler) CompletedOrders() gin.HandlerFunc {
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

		// pagination
		const (
			defaultLimit = 25
			maxLimit     = 100
		)
		limit := defaultLimit
		offset := 0
		page := 1
		if lStr := c.Query("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		if pStr := c.Query("page"); pStr != "" {
			if p, err := strconv.Atoi(pStr); err == nil && p >= 1 {
				page = p
			}
			offset = (page - 1) * limit
		} else if oStr := c.Query("offset"); oStr != "" {
			if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
				offset = o
				page = (offset / limit) + 1
			}
		}
		if limit > maxLimit {
			limit = maxLimit
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		total, err := h.orders.CountDeliveredOrdersForCustomer(ctx, customerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count delivered orders", "detail": err.Error()})
			return
		}
		list, err := h.orders.ListDeliveredOrdersForCustomer(ctx, customerID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch delivered orders", "detail": err.Error()})
			return
		}
		totalPages := 0
		if limit > 0 {
			totalPages = int((total + int64(limit) - 1) / int64(limit))
		}
		hasMore := int64(offset+len(list)) < total

		c.JSON(http.StatusOK, gin.H{
			"count":       total,
			"orders":      list,
			"limit":       limit,
			"offset":      offset,
			"page":        page,
			"total_pages": totalPages,
			"has_more":    hasMore,
		})
	}
}
