package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims carries standard and custom claims for our tokens.
type Claims struct {
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	CourierID  string `json:"courier_id,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
	AdminID    string `json:"admin_id,omitempty"`
	TokenType  string `json:"token_type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// SignJWT creates a signed JWT containing the role and profile identifiers.
func SignJWT(secret string, principal *Principal, ttl time.Duration, tokenType string) (string, error) {
	claims := Claims{
		UserID:     principal.UserID,
		Role:       principal.Role,
		CourierID:  principal.CourierID,
		CustomerID: principal.CustomerID,
		AdminID:    principal.AdminID,
		TokenType:  tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   principal.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			Issuer:    "delivery-backend",
			Audience:  jwt.ClaimStrings{tokenType},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAndValidate parses a token and validates signature and expiry.
func ParseAndValidate(secret string, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
