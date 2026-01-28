package routes

import (
	"time"

	"backend/config"
	"backend/controller"
	"backend/middleware"
	"backend/service"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter 设置所有路由
func SetupRouter(
	db *gorm.DB,
	services *config.Services,
	controllers *config.Controllers,
	repos *config.Repositories,
	cacheMiddleware *middleware.CacheMiddleware,
	sentryDSN string,
) *gin.Engine {
	router := gin.Default()

	// 全局中间件：Sentry 错误监控和恢复（使用官方 Gin SDK）
	if sentryDSN != "" {
		router.Use(sentrygin.New(sentrygin.Options{
			Repanic:         true,
			WaitForDelivery: false,
			Timeout:         2 * time.Second,
		}))
	}

	// 自定义中间件：设置用户信息到 Sentry（在 sentrygin 之后）
	router.Use(middleware.SentryUserMiddleware())

	// 健康检查端点（使用默认限流）
	router.GET("/ping", middleware.RateLimitMiddleware(middleware.DefaultRateLimiter), func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 数据库连接测试端点（使用默认限流）
	router.GET("/health", middleware.RateLimitMiddleware(middleware.DefaultRateLimiter), func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "Failed to get database instance",
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "Database connection failed",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Database connection is healthy",
		})
	})

	// API 路由组
	api := router.Group("/api")
	{
		setupAuthRoutes(api, controllers.AuthController, services.AuthService, cacheMiddleware)
		
		if controllers.ContentController != nil {
			setupContentRoutes(api, controllers.ContentController, services.AuthService, repos, cacheMiddleware)
		}

		if controllers.PurchaseController != nil {
			setupPurchaseRoutes(api, controllers.PurchaseController, services.AuthService, cacheMiddleware)
		}
	}

	return router
}

// setupAuthRoutes 设置认证路由
func setupAuthRoutes(
	api *gin.RouterGroup,
	authController *controller.AuthController,
	authService service.AuthService,
	cacheMiddleware *middleware.CacheMiddleware,
) {
	authRoutes := api.Group("/auth")
	{
		// 公开路由：注册、登录、刷新token、密码重置
		// 使用严格的限流中间件（AuthRateLimiter：5次/分钟）防止暴力破解
		authRoutes.POST("/register", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.Register)
		authRoutes.POST("/login", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.Login)
		authRoutes.POST("/refresh", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.RefreshToken)
		authRoutes.POST("/password/reset/request", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.RequestPasswordReset)
		authRoutes.POST("/password/reset", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.ResetPassword)

		// 需要认证的路由组
		// 使用默认限流（100次/分钟）和认证中间件
		protectedAuthRoutes := authRoutes.Group("")
		protectedAuthRoutes.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimiter))
		protectedAuthRoutes.Use(middleware.AuthMiddleware(authService))
		{
			// 获取当前用户信息（缓存 1 分钟，按用户区分）
			protectedAuthRoutes.GET("/me",
				cacheMiddleware.Cache(middleware.CacheOptions{
					TTL:         1 * time.Minute,
					VaryByUser:  true,
					VaryByQuery: false,
				}),
				authController.GetCurrentUser)
			// 登出当前会话
			protectedAuthRoutes.POST("/logout", authController.Logout)
			// 登出所有会话
			protectedAuthRoutes.POST("/logout-all", authController.LogoutAll)
			// 修改密码
			protectedAuthRoutes.POST("/password/change", authController.ChangePassword)
		}
	}
}

// setupContentRoutes 设置内容路由
func setupContentRoutes(
	api *gin.RouterGroup,
	contentController *controller.ContentController,
	authService service.AuthService,
	repos *config.Repositories,
	cacheMiddleware *middleware.CacheMiddleware,
) {
	contentRoutes := api.Group("/content")
	contentRoutes.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimiter))
	contentRoutes.Use(middleware.AuthMiddleware(authService))
	{
		// 批量上传视频（最多10个）- 需要 can_upload_content 权限
		contentRoutes.POST("/videos/upload",
			middleware.RequirePermission(repos.UserPermissionRepo, "can_upload_content"),
			contentController.UploadVideos)
		// 获取内容详情（缓存 5 分钟，按用户区分以支持权限检查）
		// 注意：权限检查中间件在缓存之前执行，确保只有有权限的用户才能访问缓存
		contentRoutes.GET("/:id",
			middleware.RequireNSFWPermissionForDarkContent(repos.ContentRepo, repos.UserPermissionRepo),
			cacheMiddleware.Cache(middleware.CacheOptions{
				TTL:         5 * time.Minute,
				VaryByUser:  true, // 按用户区分，因为权限不同
				VaryByQuery: false,
			}),
			contentController.GetContent)
		// 流式传输转码文件（需要权限检查 dark 内容，用于视频播放，不缓存）
		contentRoutes.GET("/files/:file_id/stream",
			middleware.RequireNSFWPermissionForDarkContentByFileID(repos.ContentFileRepo, repos.ContentRepo, repos.UserPermissionRepo),
			contentController.StreamTranscodedFile)
		// 记录内容观看（需要权限检查 dark 内容）
		contentRoutes.POST("/:id/view",
			middleware.RequireNSFWPermissionForDarkContent(repos.ContentRepo, repos.UserPermissionRepo),
			contentController.RecordContentView)
		// 获取内容分析数据（缓存 2 分钟，按查询参数区分，需要 can_view_analytics 权限）
		contentRoutes.GET("/:id/analytics",
			middleware.RequirePermission(repos.UserPermissionRepo, "can_view_analytics"),
			cacheMiddleware.Cache(middleware.CacheOptions{
				TTL:         2 * time.Minute,
				VaryByUser:  false,
				VaryByQuery: true,
			}),
			contentController.GetContentAnalytics)
	}
}

// setupPurchaseRoutes 设置购买路由
func setupPurchaseRoutes(
	api *gin.RouterGroup,
	purchaseController *controller.PurchaseController,
	authService service.AuthService,
	cacheMiddleware *middleware.CacheMiddleware,
) {
	purchaseRoutes := api.Group("/purchase")
	{
		// Webhook 路由（不需要认证，但需要 Stripe 签名验证）
		purchaseRoutes.POST("/webhook", purchaseController.HandleStripeWebhook)

		// 需要认证的路由
		protectedPurchaseRoutes := purchaseRoutes.Group("")
		protectedPurchaseRoutes.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimiter))
		protectedPurchaseRoutes.Use(middleware.AuthMiddleware(authService))
		{
			// 创建会员购买结账会话
			protectedPurchaseRoutes.POST("/membership/checkout", purchaseController.CreateMembershipCheckout)
			// 创建积分购买结账会话
			protectedPurchaseRoutes.POST("/credits/checkout", purchaseController.CreateCreditsCheckout)
			// 使用积分购买内容
			protectedPurchaseRoutes.POST("/content", purchaseController.PurchaseContent)
			// 验证结账状态
			protectedPurchaseRoutes.POST("/verify", purchaseController.VerifyCheckout)
			// 获取购买历史（缓存 1 分钟，按用户和查询参数区分）
			protectedPurchaseRoutes.GET("/history",
				cacheMiddleware.Cache(middleware.CacheOptions{
					TTL:         1 * time.Minute,
					VaryByUser:  true,
					VaryByQuery: true,
				}),
				purchaseController.GetPurchaseHistory)
		}
	}
}
