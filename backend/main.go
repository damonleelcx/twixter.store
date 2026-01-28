package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"backend/controller"
	"backend/entity"
	"backend/middleware"
	"backend/repository"
	"backend/service"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

// initRedis 初始化 Redis 连接
// 包含重试机制，适用于 Docker Compose 环境
func initRedis() (*redis.Client, error) {
	// 从环境变量获取 Redis 连接信息，如果没有则使用默认值
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	password := getEnv("REDIS_PASSWORD", "")
	dbStr := getEnv("REDIS_DB", "0")

	// 解析 Redis DB 编号
	var dbNum int
	if _, err := fmt.Sscanf(dbStr, "%d", &dbNum); err != nil {
		dbNum = 0 // 如果解析失败，使用默认值 0
	}

	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbNum,
	})

	// 测试 Redis 连接，带重试机制（适用于 Docker Compose）
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := rdb.Ping(ctx).Err()
		cancel()

		if err == nil {
			log.Println("Successfully connected to Redis")
			return rdb, nil
		}

		if i < maxRetries-1 {
			log.Printf("Failed to connect to Redis (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		} else {
			return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, err)
		}
	}

	return nil, fmt.Errorf("failed to connect to Redis")
}

// initDB 初始化数据库连接
func initDB() (*gorm.DB, error) {
	// 从环境变量获取数据库连接信息，如果没有则使用默认值
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "postgres")
	port := getEnv("DB_PORT", "5432")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// 构建 PostgreSQL 连接字符串
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)

	// 连接数据库
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 测试数据库连接
	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return database, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initPermissions 初始化系统权限
func initPermissions() {
	permissions := []entity.Permission{
		{
			Name:        "can_upload_content",
			Description: "允许上传内容",
			Resource:    "content",
			Action:      entity.ActionCreate,
			IsSystem:    true,
		},
		{
			Name:        "can_view_analytics",
			Description: "允许查看分析数据",
			Resource:    "analytics",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
		{
			Name:        "can_view_nsfw",
			Description: "允许查看NSFW内容",
			Resource:    "nsfw",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
	}

	for _, perm := range permissions {
		// 使用 FirstOrCreate 来避免重复创建
		var existing entity.Permission
		result := db.Where("name = ? OR (resource = ? AND action = ?)",
			perm.Name, perm.Resource, perm.Action).First(&existing)

		if result.Error != nil {
			// 如果权限不存在，则创建
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := db.Create(&perm).Error; err != nil {
					log.Printf("Failed to create permission %s: %v", perm.Name, err)
				} else {
					log.Printf("Created permission: %s", perm.Name)
				}
			} else {
				log.Printf("Error checking permission %s: %v", perm.Name, result.Error)
			}
		} else {
			// 权限已存在，更新描述和系统标志（如果需要）
			if existing.Description != perm.Description || existing.IsSystem != perm.IsSystem {
				existing.Description = perm.Description
				existing.IsSystem = perm.IsSystem
				if err := db.Save(&existing).Error; err != nil {
					log.Printf("Failed to update permission %s: %v", perm.Name, err)
				} else {
					log.Printf("Updated permission: %s", perm.Name)
				}
			}
		}
	}

	log.Println("Permissions initialization completed")
}

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using default values or environment variables")
	}

	// 初始化 Sentry
	sentryDSN := getEnv("SENTRY_DSN", "")
	if sentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: sentryDSN,
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				if hint.Context != nil {
					if req, ok := hint.Context.Value(sentry.RequestContextKey).(*http.Request); ok {
						// 可以在这里访问原始请求
						_ = req
					}
				}
				return event
			},
		}); err != nil {
			log.Printf("Sentry initialization failed: %v\n", err)
		} else {
			log.Println("Sentry initialized successfully")
		}
	} else {
		log.Println("Warning: SENTRY_DSN not set, Sentry monitoring disabled")
	}

	// 初始化数据库连接
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	// 自动迁移数据库表（包括分片表）
	if err := db.AutoMigrate(
		&entity.UserShard0{},
		&entity.UserShard1{},
		&entity.Session{},
		&entity.Permission{},
		&entity.UserPermission{},
		&entity.Content{},
		&entity.Tag{},
		&entity.ContentTag{},
		&entity.PurchaseShard0{},
		&entity.PurchaseShard1{},
		&entity.Wallet{},
		&entity.Analytics{},
	); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// 创建唯一索引，防止同一资源和操作的组合重复
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_resource_action_unique 
		ON permissions(resource, action) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create unique index on permissions: %v", err)
	}

	// 创建唯一索引，防止用户拥有重复的权限
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_permissions_unique 
		ON user_permissions(user_id, permission_id) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create unique index on user_permissions: %v", err)
	}

	// 创建复合索引，优化按用户ID和分片编号查询的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_permissions_user_shard 
		ON user_permissions(user_id, shard_number, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on user_permissions: %v", err)
	}

	// 创建复合索引，优化按用户ID和分片编号查询会话的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_sessions_user_shard 
		ON sessions(user_id, shard_number, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on sessions: %v", err)
	}

	// 创建复合索引，优化按上传者和分片编号查询内容的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_contents_uploaded_by_shard 
		ON contents(uploaded_by, shard_number, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on contents: %v", err)
	}

	// 创建复合索引，优化按类型和状态查询内容的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_contents_type_status 
		ON contents(type, status, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on contents (type, status): %v", err)
	}

	// 创建索引，优化按公开状态和创建时间查询内容的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_contents_public_created 
		ON contents(is_public, created_at DESC) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on contents (public, created): %v", err)
	}

	// 创建唯一索引，防止内容拥有重复的标签
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_content_tags_unique 
		ON content_tags(content_id, tag_id) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create unique index on content_tags: %v", err)
	}

	// 创建复合索引，优化按内容ID查询标签的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_content_tags_content 
		ON content_tags(content_id, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on content_tags (content): %v", err)
	}

	// 创建复合索引，优化按标签ID查询内容的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_content_tags_tag 
		ON content_tags(tag_id, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on content_tags (tag): %v", err)
	}

	// 创建复合索引，优化按用户ID查询购买记录的性能（分片0）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_user 
		ON purchases_shard_0(user_id, purchase_type, status, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on purchases_shard_0: %v", err)
	}

	// 创建复合索引，优化按用户ID查询购买记录的性能（分片1）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_user 
		ON purchases_shard_1(user_id, purchase_type, status, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on purchases_shard_1: %v", err)
	}

	// 创建索引，优化按内容ID查询购买记录的性能（分片0）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_content 
		ON purchases_shard_0(content_id, deleted_at) 
		WHERE content_id IS NOT NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on purchases_shard_0 (content): %v", err)
	}

	// 创建索引，优化按内容ID查询购买记录的性能（分片1）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_content 
		ON purchases_shard_1(content_id, deleted_at) 
		WHERE content_id IS NOT NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on purchases_shard_1 (content): %v", err)
	}

	// 创建索引，优化按交易ID查询购买记录的性能（分片0）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_transaction 
		ON purchases_shard_0(transaction_id, deleted_at) 
		WHERE transaction_id != '';
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on purchases_shard_0 (transaction): %v", err)
	}

	// 创建索引，优化按交易ID查询购买记录的性能（分片1）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_transaction 
		ON purchases_shard_1(transaction_id, deleted_at) 
		WHERE transaction_id != '';
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on purchases_shard_1 (transaction): %v", err)
	}

	// 创建索引，优化按支付网关订单ID查询购买记录的性能（分片0）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_gateway_order 
		ON purchases_shard_0(gateway_order_id, deleted_at) 
		WHERE gateway_order_id != '';
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on purchases_shard_0 (gateway_order): %v", err)
	}

	// 创建索引，优化按支付网关订单ID查询购买记录的性能（分片1）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_gateway_order 
		ON purchases_shard_1(gateway_order_id, deleted_at) 
		WHERE gateway_order_id != '';
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on purchases_shard_1 (gateway_order): %v", err)
	}

	// 创建复合索引，优化按支付网关和状态查询购买记录的性能（分片0）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_gateway_status 
		ON purchases_shard_0(payment_gateway, status, deleted_at) 
		WHERE payment_gateway != '';
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on purchases_shard_0 (gateway, status): %v", err)
	}

	// 创建复合索引，优化按支付网关和状态查询购买记录的性能（分片1）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_gateway_status 
		ON purchases_shard_1(payment_gateway, status, deleted_at) 
		WHERE payment_gateway != '';
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on purchases_shard_1 (gateway, status): %v", err)
	}

	// 创建复合索引，优化按用户ID和分片编号查询钱包的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_wallets_user_shard 
		ON wallets(user_id, shard_number, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on wallets: %v", err)
	}

	// 创建唯一约束，确保每个用户只有一个钱包（在未删除的记录中）
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_unique 
		ON wallets(user_id) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create unique index on wallets (user): %v", err)
	}

	// 创建唯一索引，防止同一内容和日期的重复记录
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_analytics_content_date_unique 
		ON analytics(content_id, date) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create unique index on analytics: %v", err)
	}

	// 创建复合索引，优化按内容ID和日期查询分析数据的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_analytics_content_date 
		ON analytics(content_id, date DESC, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index on analytics (content, date): %v", err)
	}

	// 创建索引，优化按日期范围查询分析数据的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_analytics_date 
		ON analytics(date DESC, deleted_at);
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on analytics (date): %v", err)
	}

	// 创建索引，优化按查看次数排序查询热门内容的性能
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_analytics_views 
		ON analytics(views DESC, date DESC) 
		WHERE deleted_at IS NULL;
	`).Error; err != nil {
		log.Printf("Warning: Failed to create index on analytics (views): %v", err)
	}

	log.Println("Database migration completed successfully (users_shard_0, users_shard_1, sessions, permissions, user_permissions, contents, tags, content_tags, purchases_shard_0, purchases_shard_1, wallets, analytics)")

	// 初始化系统权限
	initPermissions()

	// 初始化 Redis 连接
	redisClient, err := initRedis()
	if err != nil {
		log.Fatalf("Redis initialization failed: %v", err)
	}
	defer redisClient.Close()

	// 初始化 Redis 限流器
	middleware.DefaultRateLimiter = middleware.NewRedisRateLimiter(redisClient, 100, 1*time.Minute)
	middleware.StrictRateLimiter = middleware.NewRedisRateLimiter(redisClient, 10, 1*time.Minute)
	middleware.AuthRateLimiter = middleware.NewRedisRateLimiter(redisClient, 5, 1*time.Minute)

	// 初始化 repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	userPermissionRepo := repository.NewUserPermissionRepository(db) // 用于权限中间件，已准备好供将来使用

	// 初始化 services
	authService := service.NewAuthService(db, userRepo, sessionRepo)

	// 初始化 controllers
	authController := controller.NewAuthController(authService)

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
		// 认证路由组
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
				// 获取当前用户信息
				protectedAuthRoutes.GET("/me", authController.GetCurrentUser)
				// 登出当前会话
				protectedAuthRoutes.POST("/logout", authController.Logout)
				// 登出所有会话
				protectedAuthRoutes.POST("/logout-all", authController.LogoutAll)
				// 修改密码
				protectedAuthRoutes.POST("/password/change", authController.ChangePassword)
			}
		}

		// 其他需要认证的API路由可以在这里添加
		// 例如：
		// protectedRoutes := api.Group("")
		// protectedRoutes.Use(middleware.RateLimitMiddleware(middleware.DefaultRateLimiter))
		// protectedRoutes.Use(middleware.AuthMiddleware(authService))
		// {
		//     // 需要特定权限的路由
		//     contentRoutes := protectedRoutes.Group("/content")
		//     contentRoutes.Use(middleware.RequirePermission(userPermissionRepo, "can_upload_content"))
		//     {
		//         contentRoutes.POST("/upload", contentController.Upload)
		//     }
		// }
		_ = userPermissionRepo // 保留供将来使用权限中间件
	}

	router.Run() // listens on 0.0.0.0:8080 by default
}
