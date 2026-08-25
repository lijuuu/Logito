// Package auth validates the jwt tokens issued by query-interface's /auth/login.
// log-ingestor does not issue tokens itself, only checks them, using the same
// shared secret (config Auth.JWTSecret) both services are configured with.
package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lijuuu/Logito/log-ingestor/internal/config"
)

type Role string

const (
	RoleViewer Role = "viewer"
	RoleAdmin  Role = "admin"
)

type Claims struct {
	Email string `json:"email"`
	Role  Role   `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	jwtSecret []byte
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		jwtSecret: []byte(cfg.Auth.JWTSecret),
	}
}

func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
