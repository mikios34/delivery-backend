package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	authpkg "github.com/mikios34/delivery-backend/auth"
)

// RequireAuth validates Bearer JWT, places claims into context and continues.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}
		tokenString := authHeader[7:]

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "dev-insecure-secret-change-me"
		}

		claims := &authpkg.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		if claims.CourierID != "" {
			c.Set("courier_id", claims.CourierID)
		}
		if claims.CustomerID != "" {
			c.Set("customer_id", claims.CustomerID)
		}
		if claims.AdminID != "" {
			c.Set("admin_id", claims.AdminID)
		}
		c.Next()
	}
}

// RequireRoles ensures the authenticated principal has one of the allowed roles.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	roleSet := map[string]struct{}{}
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role := c.GetString("role")
		if _, ok := roleSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: insufficient role"})
			return
		}	
		c.Next()
	}
}
