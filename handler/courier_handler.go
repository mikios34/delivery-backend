package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authpkg "github.com/mikios34/delivery-backend/auth"
	courierSvc "github.com/mikios34/delivery-backend/courier"
	"github.com/mikios34/delivery-backend/entity"
	orderpkg "github.com/mikios34/delivery-backend/order"
)

// CourierHandler bundles dependencies for courier-related HTTP handlers.
//
// Note: Firebase verification is intentionally omitted here because you said
// Firebase authentication will be handled in the frontend. The frontend should
// include a trusted identifier (e.g. firebase_uid) in the registration payload.
type CourierHandler struct {
	service courierSvc.CourierService
	orders  orderpkg.Repository
}

// NewCourierHandler constructs a CourierHandler.
func NewCourierHandler(svc courierSvc.CourierService) *CourierHandler {
	return &CourierHandler{service: svc}
}

// WithOrders injects the order repository for active order lookup.
func (h *CourierHandler) WithOrders(orders orderpkg.Repository) *CourierHandler {
	h.orders = orders
	return h
}

// payload for POST /api/v1/couriers/register
type registerCourierPayload struct {
	FirstName        string `json:"first_name" binding:"required"`
	LastName         string `json:"last_name" binding:"required"`
	Phone            string `json:"phone" binding:"required"`
	HasVehicle       bool   `json:"has_vehicle"`
	PrimaryVehicle   string `json:"primary_vehicle"`
	VehicleDetails   string `json:"vehicle_details"`
	GuarantyOptionID string `json:"guaranty_option_id" binding:"required"` // UUID string
	FirebaseUID      string `json:"firebase_uid" binding:"required"`       // provided by frontend after Firebase auth
	ProfilePicture   string `json:"profile_picture"`                       // optional profile picture URL
}

// RegisterCourier registers a courier (creates user, courier, guaranty payment placeholder).
// The frontend is responsible for Firebase authentication and must provide firebase_uid
// in the payload. The backend trusts that value per your instruction.
func (h *CourierHandler) RegisterCourier() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p registerCourierPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}

		// parse guaranty option id
		guarID, err := uuid.Parse(p.GuarantyOptionID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guaranty_option_id", "detail": err.Error()})
			return
		}

		// prepare service request (firebase uid provided by frontend)
		req := courierSvc.RegisterCourierRequest{
			FirstName:        p.FirstName,
			LastName:         p.LastName,
			Phone:            p.Phone,
			FirebaseUID:      p.FirebaseUID,
			HasVehicle:       p.HasVehicle,
			PrimaryVehicle:   entity.VehicleType(p.PrimaryVehicle),
			VehicleDetails:   p.VehicleDetails,
			GuarantyOptionID: guarID,
			ProfilePicture:   p.ProfilePicture,
		}

		// call service with request context
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		createdCourier, err := h.service.RegisterCourier(ctx, req)
		if err != nil {
			// best-effort error mapping
			switch err.Error() {
			case "courier with this phone already exists":
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			case "selected guaranty option not found or inactive", "guaranty option not found or not active":
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register courier", "detail": err.Error()})
			}
			return
		}

		// Build principal-like response and sign JWT for immediate use
		principal := authpkg.Principal{
			UserID:    createdCourier.UserID.String(),
			CourierID: createdCourier.ID.String(),
			// Role omitted from JSON, but included in JWT claims
			Role:      "courier",
			FirstName: p.FirstName,
			LastName:  p.LastName,
			Phone:     p.Phone,
		}
		if p.ProfilePicture != "" {
			pp := p.ProfilePicture
			principal.ProfilePicture = &pp
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

// ListGuarantyOptions returns active guaranty options for the signup dropdown.

// SetAvailability toggles courier availability (requires auth + courier role on route).
func (h *CourierHandler) SetAvailability() gin.HandlerFunc {
	type payload struct {
		Available bool `json:"available"`
	}
	return func(c *gin.Context) {
		var p payload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		courierIDStr := c.GetString("courier_id")
		if courierIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "courier_id missing in context"})
			return
		}
		id, err := uuid.Parse(courierIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid courier_id in token"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		if err := h.service.SetAvailability(ctx, id, p.Available); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update availability", "detail": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// UpdateLocation updates courier location (lat/lng) (requires auth + courier role on route).
func (h *CourierHandler) UpdateLocation() gin.HandlerFunc {
	type payload struct {
		CourierID string   `json:"courier_id" binding:"required"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	return func(c *gin.Context) {
		var p payload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		id, err := uuid.Parse(p.CourierID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid courier_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		if err := h.service.UpdateLocation(ctx, id, p.Latitude, p.Longitude); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update location", "detail": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// GET /api/v1/guaranty-options
func (h *CourierHandler) ListGuarantyOptions() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		opts, err := h.service.ListGuarantyOptions(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list guaranty options", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, opts)
	}
}

// ActiveOrder returns the courier's current active order (status not in no_nearby_driver, delivered)
func (h *CourierHandler) ActiveOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.orders == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "orders repository not configured"})
			return
		}
		courierIDStr := c.GetString("courier_id")
		if courierIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "courier_id missing in context"})
			return
		}
		courierID, err := uuid.Parse(courierIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid courier_id"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		ord, err := h.orders.GetActiveOrderForCourier(ctx, courierID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if ord == nil {
			c.JSON(http.StatusOK, gin.H{"active": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"active": true, "order": ord})
	}
}
