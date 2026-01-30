package routes

import (
	"os"
	"strings"
	"time"

	"backend/config"
	"backend/controller"
	"backend/middleware"
	"backend/service"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 视频上传等 multipart 最大内存（100GB），等效不限制
const maxMultipartMemory = 100 << 30

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
	router.MaxMultipartMemory = maxMultipartMemory

	// CORS：当前端通过 NEXT_PUBLIC_API_URL 直连后端时（如 localhost:3000 → localhost:8080）需允许跨域
	origins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
	}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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

		tagsController := controller.NewTagsController(repos.TagRepo, repos.UserPermissionRepo)
		setupTagsRoutes(api, tagsController, services.AuthService, repos)

		if controllers.PurchaseController != nil {
			setupPurchaseRoutes(api, controllers.PurchaseController, services.AuthService, cacheMiddleware)
		}

		if controllers.AnalyticsController != nil {
			setupAdminAnalyticsRoutes(api, controllers.AnalyticsController, services.AuthService, repos, cacheMiddleware)
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
		// 登录/注册等使用严格限流（5次/分钟）防暴力破解；refresh 使用宽松限流（30次/分钟）避免多标签/重试时 429
		authRoutes.POST("/register", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.Register)
		authRoutes.POST("/login", middleware.RateLimitMiddleware(middleware.AuthRateLimiter), authController.Login)
		authRoutes.POST("/refresh", middleware.RateLimitMiddleware(middleware.RefreshRateLimiter), authController.RefreshToken)
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
	contentRoutes.Use(middleware.ViewingCookieMiddleware()) // 解码 ?viewing= 并设置 cookie twixter_viewing
	// 列表 feed（可选登录；category=dark 时需 can_view_nsfw 或 cookie viewing_dark）
	contentRoutes.GET("/list",
		middleware.OptionalAuthMiddleware(authService),
		middleware.RequireNSFWPermissionForDarkList(repos.UserPermissionRepo),
		contentController.ListContent)
	// GIF 预览：已购买返回原图，未购买返回后端模糊后的 JPEG（可选登录）
	contentRoutes.GET("/files/:file_id/preview",
		middleware.OptionalAuthMiddleware(authService),
		contentController.StreamGifPreview)
	// 字面路径 /library、/bookmarks、/viewing-token 必须在 /:id 之前注册
	contentRoutes.GET("/library", middleware.AuthMiddleware(authService), contentController.ListLibrary)
	contentRoutes.GET("/bookmarks", middleware.AuthMiddleware(authService), contentController.ListBookmarks)
	contentRoutes.GET("/viewing-token", contentController.GetViewingToken)
	// 内容详情与流：可选登录，未登录时凭 cookie viewing_dark 可访问 dark 内容
	contentRoutes.GET("/:id",
		middleware.OptionalAuthMiddleware(authService),
		middleware.RequireNSFWPermissionForDarkContent(repos.ContentRepo, repos.UserPermissionRepo),
		cacheMiddleware.Cache(middleware.CacheOptions{
			TTL:         5 * time.Minute,
			VaryByUser:  true,
			VaryByQuery: false,
		}),
		contentController.GetContent)
	contentRoutes.GET("/files/:file_id/stream",
		middleware.OptionalAuthMiddleware(authService),
		middleware.RequireNSFWPermissionForDarkContentByFileID(repos.ContentFileRepo, repos.ContentRepo, repos.UserPermissionRepo),
		contentController.StreamTranscodedFile)
	// 记录内容观看：可选登录；未登录时仅跳过记录，不返回 401（与 viewing cookie 访问一致）
	contentRoutes.POST("/:id/view",
		middleware.OptionalAuthMiddleware(authService),
		middleware.RequireNSFWPermissionForDarkContent(repos.ContentRepo, repos.UserPermissionRepo),
		contentController.RecordContentView)
	// 记录观看进度（需登录，用于更新 TotalWatchTime / AverageWatchTime / CompletionRate）
	contentRoutes.POST("/:id/watch-progress",
		middleware.AuthMiddleware(authService),
		middleware.RequireNSFWPermissionForDarkContent(repos.ContentRepo, repos.UserPermissionRepo),
		contentController.RecordWatchProgress)

	contentAuthRoutes := contentRoutes.Group("")
	contentAuthRoutes.Use(middleware.AuthMiddleware(authService))
	{
		// 批量上传视频（最多10个）- 需要 admin 账户类型 + can_upload_content 权限
		contentAuthRoutes.POST("/videos/upload",
			middleware.RequirePermission(repos.UserPermissionRepo, "can_upload_content"),
			contentController.UploadVideos)
		// 更新内容元数据（需 can_edit_content 权限，仅查 user_permissions）
		contentAuthRoutes.PATCH("/:id",
			middleware.RequirePermission(repos.UserPermissionRepo, "can_edit_content"),
			contentController.UpdateContent)
		// 书签：添加 / 移除
		contentAuthRoutes.POST("/:id/bookmark", contentController.AddBookmark)
		contentAuthRoutes.DELETE("/:id/bookmark", contentController.RemoveBookmark)
		// 获取内容分析数据（缓存 2 分钟，按查询参数区分，需要 can_view_analytics 权限）
		contentAuthRoutes.GET("/:id/analytics",
			middleware.RequirePermission(repos.UserPermissionRepo, "can_view_analytics"),
			cacheMiddleware.Cache(middleware.CacheOptions{
				TTL:         2 * time.Minute,
				VaryByUser:  false,
				VaryByQuery: true,
			}),
			contentController.GetContentAnalytics)
	}
}

// setupTagsRoutes 设置标签路由
func setupTagsRoutes(
	api *gin.RouterGroup,
	tagsController *controller.TagsController,
	authService service.AuthService,
	repos *config.Repositories,
) {
	tagsRoutes := api.Group("/tags")
	tagsRoutes.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimiter))
	// 热门标签（可选登录；category=all 需 can_view_nsfw，在 controller 内校验）
	tagsRoutes.GET("/trending",
		middleware.OptionalAuthMiddleware(authService),
		tagsController.TrendTags)
	tagsProtected := tagsRoutes.Group("")
	tagsProtected.Use(middleware.AuthMiddleware(authService))
	{
		// 搜索标签（自动完成）- 需 can_search_tags；admin 和 dark 账户类型默认拥有
		tagsProtected.GET("/search",
			middleware.RequirePermission(repos.UserPermissionRepo, "can_search_tags"),
			tagsController.SearchTags)
	}
}

// setupAdminAnalyticsRoutes 设置管理员分析路由（需 can_view_analytics）
func setupAdminAnalyticsRoutes(
	api *gin.RouterGroup,
	analyticsController *controller.AnalyticsController,
	authService service.AuthService,
	repos *config.Repositories,
	cacheMiddleware *middleware.CacheMiddleware,
) {
	adminAnalytics := api.Group("/admin/analytics")
	adminAnalytics.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimiter))
	adminAnalytics.Use(middleware.AuthMiddleware(authService))
	adminAnalytics.Use(middleware.RequirePermission(repos.UserPermissionRepo, "can_view_analytics"))
	{
		adminAnalytics.GET("/video",
			cacheMiddleware.Cache(middleware.CacheOptions{
				TTL:         2 * time.Minute,
				VaryByUser:  false,
				VaryByQuery: true,
			}),
			analyticsController.GetAdminVideoAnalytics)
		adminAnalytics.GET("/revenue",
			cacheMiddleware.Cache(middleware.CacheOptions{
				TTL:         2 * time.Minute,
				VaryByUser:  false,
				VaryByQuery: true,
			}),
			analyticsController.GetAdminRevenueAnalytics)
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
			// 创建会员购买结账会话（Stripe）
			protectedPurchaseRoutes.POST("/membership/checkout", purchaseController.CreateMembershipCheckout)
			// 创建会员购买 PayPal 订单（备用支付）
			protectedPurchaseRoutes.POST("/membership/paypal/order", purchaseController.CreatePayPalMembershipOrder)
			protectedPurchaseRoutes.POST("/membership/paypal/capture", purchaseController.CapturePayPalOrder)
			// 创建积分购买结账会话（Stripe）
			protectedPurchaseRoutes.POST("/credits/checkout", purchaseController.CreateCreditsCheckout)
			// 创建积分购买 PayPal 订单（备用支付）
			protectedPurchaseRoutes.POST("/credits/paypal/order", purchaseController.CreatePayPalCreditsOrder)
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
