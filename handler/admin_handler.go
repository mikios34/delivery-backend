package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	adminpkg "github.com/mikios34/delivery-backend/admin"
)

// AdminHandler bundles dependencies for admin-related HTTP handlers.
type AdminHandler struct {
	service adminpkg.AdminService
}

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler(svc adminpkg.AdminService) *AdminHandler {
	return &AdminHandler{service: svc}
}

type registerAdminPayload struct {
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	FirebaseUID string `json:"firebase_uid" binding:"required"`
}

// RegisterAdmin registers an admin (creates user and admin profile).
func (h *AdminHandler) RegisterAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p registerAdminPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}

		req := adminpkg.RegisterAdminRequest{
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			Phone:       p.Phone,
			FirebaseUID: p.FirebaseUID,
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		createdAdmin, err := h.service.RegisterAdmin(ctx, req)
		if err != nil {
			switch err.Error() {
			case "admin with this phone already exists":
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register admin", "detail": err.Error()})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "admin created; phone verified (frontend)",
			"admin": gin.H{
				"id":      createdAdmin.ID,
				"user_id": createdAdmin.UserID,
			},
		})
	}
}
