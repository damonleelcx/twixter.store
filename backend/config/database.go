package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// InitDB 初始化数据库连接
func InitDB() (*gorm.DB, error) {
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

// InitRedis 初始化 Redis 连接
// 包含重试机制，适用于 Docker Compose 环境
func InitRedis() (*redis.Client, error) {
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
