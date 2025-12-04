package api

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	adminpkg "github.com/mikios34/delivery-backend/admin"
	authpkg "github.com/mikios34/delivery-backend/auth"
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

		// Return principal for consistency with login response
		principal := authpkg.Principal{
			UserID:    createdAdmin.UserID.String(),
			AdminID:   createdAdmin.ID.String(),
			Role:      "admin",
			FirstName: p.FirstName,
			LastName:  p.LastName,
			Phone:     p.Phone,
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
