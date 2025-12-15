package middleware

import (
	"net/http"

	fbAuth "firebase.google.com/go/auth"
	"github.com/gin-gonic/gin"
)

// RequireFirebaseAuth validates a Firebase ID token (Bearer) and sets
// `firebase_uid` (and optionally `role` if present as a custom claim) in context.
//
// Typical usage:
//
//	mw.RequireFirebaseAuth(firebaseAuthClient)
func RequireFirebaseAuth(client *fbAuth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "firebase auth not configured"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			return
		}
		idToken := authHeader[7:]

		token, err := client.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired firebase token"})
			return
		}

		c.Set("firebase_uid", token.UID)
		if role, ok := token.Claims["role"].(string); ok && role != "" {
			c.Set("role", role)
		}
		c.Next()
	}
}
