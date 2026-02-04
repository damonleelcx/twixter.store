package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	// 构建 PostgreSQL 连接字符串；connect_timeout 避免拨号长时间挂起（K3s/云环境 connection refused 时快速失败便于重试）
	connectTimeout := getEnv("DB_CONNECT_TIMEOUT", "5")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s connect_timeout=%s",
		host, user, password, dbname, port, sslmode, connectTimeout)

	maxRetries := 10
	retryDelay := 2 * time.Second

	// GORM logger: 不把 "record not found" 当错误打印（业务层正常处理，避免登录/列表等请求刷屏）
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			IgnoreRecordNotFoundError: true,
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			Colorful:                  false,
		},
	)

	var database *gorm.DB
	var err error
	for i := 0; i < maxRetries; i++ {
		database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
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
				maxIdle := 5
				maxOpen := 25
				if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n >= 0 {
						maxIdle = n
					}
				}
				if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						maxOpen = n
					}
				}
				sqlDB.SetMaxIdleConns(maxIdle)
				sqlDB.SetMaxOpenConns(maxOpen)
				sqlDB.SetConnMaxLifetime(5 * time.Minute) // 定期回收连接，减少使用已断开的连接
				connMaxIdleTime := 30 * time.Second       // 默认 30s：早于常见防火墙空闲超时，避免复用被关闭的连接导致 fetch 时 connection refused
				if v := os.Getenv("DB_CONN_MAX_IDLE_TIME"); v != "" {
					if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
						connMaxIdleTime = time.Duration(sec) * time.Second
					}
				}
				sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
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
