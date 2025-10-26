package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lijuuu/Logito/query-interface/internal/config"
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
		jwtSecret: []byte(cfg.QueryInterface.Auth.Password), // Using password as JWT secret
	}
}

func (a *AuthService) GenerateToken(email string, role Role) (string, error) {
	claims := &Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
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

func (a *AuthService) AuthenticateUser(email, password string, cfg *config.Config) (Role, error) {
	// Only allow admin authentication
	if email == cfg.QueryInterface.Auth.Email && password == cfg.QueryInterface.Auth.Password {
		return RoleAdmin, nil
	}

	return "", errors.New("invalid credentials")
}

func (a *AuthService) HasPermission(role Role, action string) bool {
	// Only admin role is allowed for all actions
	return role == RoleAdmin
}
