package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	fbAuth "firebase.google.com/go/auth"
	"github.com/gin-gonic/gin"
	authpkg "github.com/mikios34/delivery-backend/auth"
	"gorm.io/gorm"
)

type AuthHandler struct {
	service      authpkg.Service
	firebaseAuth *fbAuth.Client
}

func NewAuthHandler(svc authpkg.Service) *AuthHandler { return &AuthHandler{service: svc} }

// WithFirebaseAuth injects a Firebase Admin auth client for token verification.
func (h *AuthHandler) WithFirebaseAuth(client *fbAuth.Client) *AuthHandler {
	h.firebaseAuth = client
	return h
}

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

// ExchangeFirebase verifies a Firebase ID token and issues backend JWTs.
// Request body: { "id_token": "<firebase-id-token>" }
type exchangePayload struct {
	IDToken string `json:"id_token"`
}

func (h *AuthHandler) ExchangeFirebase() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.firebaseAuth == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "firebase auth not configured"})
			return
		}
		var p exchangePayload
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "detail": err.Error()})
			return
		}
		if p.IDToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id_token is required"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		token, err := h.firebaseAuth.VerifyIDToken(ctx, p.IDToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid firebase token", "detail": err.Error()})
			return
		}
		// Use the Firebase UID to perform application login and issue backend JWTs.
		req := authpkg.LoginRequest{FirebaseUID: token.UID}
		principal, err := h.service.Login(ctx, req)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":        "user not registered",
					"firebase_uid": token.UID,
				})
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "login failed", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"principal": principal})
	}
}
