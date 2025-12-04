package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	authpkg "github.com/mikios34/delivery-backend/auth"
)

type AuthHandler struct {
	service authpkg.Service
}

func NewAuthHandler(svc authpkg.Service) *AuthHandler { return &AuthHandler{service: svc} }

type loginPayload struct {
	Phone       string `json:"phone"`
	FirebaseUID string `json:"firebase_uid"`
}

func (h *AuthHandler) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p loginPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		if p.FirebaseUID == "" && p.Phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "either firebase_uid or phone is required"})
			return
		}
		req := authpkg.LoginRequest{Phone: p.Phone, FirebaseUID: p.FirebaseUID}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		principal, err := h.service.Login(ctx, req)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login failed", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"principal": principal})
	}
}

type refreshPayload struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		var p refreshPayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		if p.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		principal, err := h.service.Refresh(ctx, p.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh failed", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"principal": principal})
	}
}
