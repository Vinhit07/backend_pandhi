package utils

import (
	"backend_pandhi/pkg/config"
	"backend_pandhi/pkg/models"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims represents the custom JWT claims
type TokenClaims struct {
	ID    int         `json:"id"`
	Email string      `json:"email"`
	Role  models.Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID int, email string, role models.Role) (string, error) {
	// Parse expiration duration (default 7d)
	expiresIn := config.AppConfig.JWTExpiresIn
	var duration time.Duration
	
	// Simple duration parsing (matching Express.js behavior)
	switch expiresIn {
	case "7d":
		duration = 7 * 24 * time.Hour
	case "1d":
		duration = 24 * time.Hour
	case "30m":
		duration = 30 * time.Minute
	default:
		duration = 7 * 24 * time.Hour // Default to 7 days
	}

	// Create claims
	claims := TokenClaims{
		ID:    userID,
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// VerifyToken verifies and parses a JWT token
func VerifyToken(tokenString string) (*TokenClaims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims
	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetTokenExpiration returns the expiration time of a token
func GetTokenExpiration(tokenString string) (time.Time, error) {
	claims, err := VerifyToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}
