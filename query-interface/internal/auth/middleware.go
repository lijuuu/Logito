package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// Require authentication for all routes
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

func RequireRole(requiredRole Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(Role)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid role type",
			})
			c.Abort()
			return
		}

		// Admin can access everything, viewers have limited access
		if requiredRole == RoleAdmin && role != RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Admin role required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequirePermission(authService *AuthService, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User role not found",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(Role)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid role type",
			})
			c.Abort()
			return
		}

		if !authService.HasPermission(role, action) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAuth is now redundant since AuthMiddleware requires authentication for all routes
// Keeping for backward compatibility but it's essentially a no-op now
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
