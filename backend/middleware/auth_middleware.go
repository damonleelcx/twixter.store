package middleware

import (
	"backend/entity"
	"backend/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthTokenCookieName 与前端 AUTH_TOKEN_COOKIE 一致，用于 GET 请求（如 img src 预览）带 cookie 时识别登录用户
const AuthTokenCookieName = "twixter_token"

// AuthMiddleware Authentication middleware
func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token is required",
			})
			c.Abort()
			return
		}

		// Validate token
		session, user, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Check if account is locked
		locked, err := authService.CheckAccountLocked(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check account status",
			})
			c.Abort()
			return
		}

		if locked {
			c.JSON(http.StatusLocked, gin.H{
				"error": "Account is locked",
			})
			c.Abort()
			return
		}

		// Store session and user in context
		c.Set("session", session)
		c.Set("user", user)

		c.Next()
	}
}

// OptionalAuthMiddleware Optional authentication middleware (doesn't fail if no token).
// 先读 Authorization 头，若无则读 cookie twixter_token，以便 GET 请求（如 img src 预览）带 cookie 时能识别会员/已购买并返回正常 GIF。
func OptionalAuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" {
				token = parts[1]
			}
		}
		if token == "" {
			token, _ = c.Cookie(AuthTokenCookieName)
		}
		if token == "" {
			c.Next()
			return
		}

		// Validate token (silently fail if invalid)
		session, user, err := authService.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		// Store session and user in context if valid
		c.Set("session", session)
		c.Set("user", user)

		c.Next()
	}
}

// GetUserFromContext Get user from context
func GetUserFromContext(c *gin.Context) (*entity.UserBase, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	userBase, ok := user.(*entity.UserBase)
	return userBase, ok
}

// GetSessionFromContext Get session from context
func GetSessionFromContext(c *gin.Context) (*entity.Session, bool) {
	session, exists := c.Get("session")
	if !exists {
		return nil, false
	}

	sess, ok := session.(*entity.Session)
	return sess, ok
}
