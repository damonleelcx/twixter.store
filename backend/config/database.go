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

// InitDB 初始化数据库连接，带重试（适用于 k8s 中 DB 晚于 Pod 就绪的场景）
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

	maxRetries := 10
	retryDelay := 2 * time.Second

	var database *gorm.DB
	var err error
	for i := 0; i < maxRetries; i++ {
		database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, dbErr := database.DB()
			if dbErr != nil {
				database = nil
				err = dbErr
			} else if pingErr := sqlDB.Ping(); pingErr != nil {
				sqlDB.Close()
				database = nil
				err = pingErr
			} else {
				// 配置连接池，避免长时间空闲连接被服务端/防火墙关闭后仍被复用导致 connection refused
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetConnMaxLifetime(5 * time.Minute) // 定期回收连接，减少使用已断开的连接
				sqlDB.SetConnMaxIdleTime(1 * time.Minute) // 空闲超过 1 分钟即关闭，避免 K3s/云环境防火墙关闭空闲连接后复用导致 dial tcp: connection refused
				log.Println("Successfully connected to PostgreSQL database")
				return database, nil
			}
		}
		if i < maxRetries-1 {
			log.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		} else {
			return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
		}
	}

	return nil, fmt.Errorf("failed to connect to database")
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
