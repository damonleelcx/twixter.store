package main

import (
	"log"
	"os"

	"backend/config"
	"backend/database"
	"backend/routes"

	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using default values or environment variables")
	}

	// 初始化 Sentry
	config.InitSentry()
	sentryDSN := config.GetSentryDSN()

	// 初始化数据库连接
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	// 运行数据库迁移和创建索引
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// 初始化系统权限
	config.InitPermissions(db)

	// 初始化 Redis 连接
	redisClient, err := config.InitRedis()
	if err != nil {
		log.Fatalf("Redis initialization failed: %v", err)
	}
	defer redisClient.Close()

	// 初始化中间件
	cacheMiddleware := config.InitMiddleware(redisClient)

	// 初始化 repositories
	repos := config.InitRepositories(db)

	// 初始化 services
	services, err := config.InitServices(db, repos)
	if err != nil {
		log.Fatalf("Services initialization failed: %v", err)
	}

	// 若不存在任何 admin 则根据环境变量 ADMIN_EMAIL、ADMIN_PASSWORD 创建种子 admin
	if err := services.AuthService.EnsureAdminSeed(os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Printf("Warning: EnsureAdminSeed failed: %v", err)
	}

	// 初始化 controllers
	controllers := config.InitControllers(services, repos)

	// 设置路由
	router := routes.SetupRouter(db, services, controllers, repos, cacheMiddleware, sentryDSN)

	// 启动服务器
	router.Run() // listens on 0.0.0.0:8080 by default
}
