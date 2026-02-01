package main

// This is an example file showing how to use the auth controller and middlewares
// DO NOT include this file in your actual build

/*
import (
	"backend/controller"
	"backend/middleware"
	"backend/repository"
	"backend/service"

	"github.com/gin-gonic/gin"
)

func setupAuthRoutes(router *gin.Engine, db *gorm.DB) {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Initialize service
	authService := service.NewAuthService(db, userRepo, sessionRepo)

	// Initialize controller
	authController := controller.NewAuthController(authService)

	// Create auth routes group
	authRoutes := router.Group("/api/auth")
	{
		// Public routes with rate limiting
		authRoutes.POST("/register", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.Register)
		authRoutes.POST("/login", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.Login)
		authRoutes.POST("/refresh", middleware.RateLimitMiddleware(middleware.DefaultRateLimiter), authController.RefreshToken)
		authRoutes.POST("/password/reset/request", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.RequestPasswordReset)
		authRoutes.POST("/password/reset", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.ResetPassword)

		// Protected routes (require authentication)
		authRoutes.Use(middleware.AuthMiddleware(authService))
		{
			authRoutes.GET("/me", authController.GetCurrentUser)
			authRoutes.POST("/logout", authController.Logout)
			authRoutes.POST("/logout-all", authController.LogoutAll)
			authRoutes.POST("/password/change", authController.ChangePassword)
		}
	}

	// Example: Protected routes with authentication
	apiRoutes := router.Group("/api")
	apiRoutes.Use(middleware.AuthMiddleware(authService))
	{
		// All routes under /api require authentication
		apiRoutes.GET("/protected", func(c *gin.Context) {
			user, _ := middleware.GetUserFromContext(c)
			c.JSON(200, gin.H{
				"message": "This is a protected route",
				"user_id": user.ID,
			})
		})

		// Admin only routes
		adminRoutes := apiRoutes.Group("/admin")
		adminRoutes.Use(middleware.RequireAdmin())
		{
			adminRoutes.GET("/dashboard", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Admin dashboard",
				})
			})
		}
	}
}
*/
