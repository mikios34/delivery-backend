package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	courierSvc "github.com/mikios34/delivery-backend/courier"
	"github.com/mikios34/delivery-backend/entity"
)

// CourierHandler bundles dependencies for courier-related HTTP handlers.
//
// Note: Firebase verification is intentionally omitted here because you said
// Firebase authentication will be handled in the frontend. The frontend should
// include a trusted identifier (e.g. firebase_uid) in the registration payload.
type CourierHandler struct {
	service courierSvc.CourierService
}

// NewCourierHandler constructs a CourierHandler.
func NewCourierHandler(svc courierSvc.CourierService) *CourierHandler {
	return &CourierHandler{service: svc}
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

		// return minimal courier info
		c.JSON(http.StatusCreated, gin.H{
			"message": "courier created; phone verified (frontend)",
			"courier": gin.H{
				"id":              createdCourier.ID,
				"user_id":         createdCourier.UserID,
				"guaranty_paid":   createdCourier.GuarantyPaid,
				"primary_vehicle": createdCourier.PrimaryVehicle,
			},
		})
	}
}

// ListGuarantyOptions returns active guaranty options for the signup dropdown.
// GET /api/v1/guaranty-options
func (h *CourierHandler) ListGuarantyOptions() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		opts, err := h.service.ListGuarantyOptions(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list guaranty options", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, opts)
	}
}
