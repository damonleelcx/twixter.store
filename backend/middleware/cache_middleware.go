package middleware

import (
	"backend/entity"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// CacheMiddleware Redis 缓存中间件
type CacheMiddleware struct {
	client     *redis.Client
	prefix     string
	defaultTTL time.Duration
}

// NewCacheMiddleware 创建新的缓存中间件
func NewCacheMiddleware(client *redis.Client, prefix string, defaultTTL time.Duration) *CacheMiddleware {
	if prefix == "" {
		prefix = "cache:"
	}
	return &CacheMiddleware{
		client:     client,
		prefix:     prefix,
		defaultTTL: defaultTTL,
	}
}

// CacheOptions 缓存选项
type CacheOptions struct {
	TTL         time.Duration             // 缓存过期时间，0 表示使用默认值
	KeyFunc     func(*gin.Context) string // 自定义缓存键生成函数
	SkipCache   func(*gin.Context) bool   // 是否跳过缓存
	VaryByUser  bool                      // 是否按用户区分缓存
	VaryByQuery bool                      // 是否按查询参数区分缓存
}

// Cache 缓存中间件
func (cm *CacheMiddleware) Cache(options CacheOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只缓存 GET 请求
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// 检查是否跳过缓存
		if options.SkipCache != nil && options.SkipCache(c) {
			c.Next()
			return
		}

		// 生成缓存键
		var cacheKey string
		if options.KeyFunc != nil {
			cacheKey = options.KeyFunc(c)
		} else {
			cacheKey = cm.generateCacheKey(c, options)
		}

		// 尝试从缓存获取
		ctx := context.Background()
		cachedData, err := cm.client.Get(ctx, cacheKey).Result()
		if err == nil {
			// 缓存命中，直接返回
			c.Header("X-Cache", "HIT")
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(cachedData))
			c.Abort()
			return
		}

		// 缓存未命中，继续处理请求
		c.Header("X-Cache", "MISS")

		// 使用自定义响应写入器捕获响应
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// 处理请求
		c.Next()

		// 只缓存成功的响应（2xx 状态码）
		if writer.Status() >= 200 && writer.Status() < 300 {
			// 获取响应体
			responseBody := writer.body.String()
			// 超过此大小不写入 Redis，避免 "Single item size exceeds maxSize" 等限制
			const maxCacheBodyBytes = 512 * 1024 // 512KB
			if len(responseBody) > maxCacheBodyBytes {
				// 跳过缓存，避免超大响应撑爆 Redis
				return
			}

			// 确定 TTL
			ttl := options.TTL
			if ttl == 0 {
				ttl = cm.defaultTTL
			}

			// 存储到缓存
			if ttl > 0 {
				_ = cm.client.Set(ctx, cacheKey, responseBody, ttl).Err()
			}
		}
	}
}

// generateCacheKey 生成缓存键
func (cm *CacheMiddleware) generateCacheKey(c *gin.Context, options CacheOptions) string {
	parts := []string{
		c.Request.Method,
		c.Request.URL.Path,
	}

	// 如果按用户区分，添加用户ID
	if options.VaryByUser {
		if user, exists := c.Get("user"); exists {
			// 尝试从 UserBase 获取 ID
			if userBase, ok := user.(*entity.UserBase); ok {
				parts = append(parts, fmt.Sprintf("user:%d", userBase.ID))
			} else if userBase, ok := user.(interface{ GetID() uint }); ok {
				// 兼容其他可能的用户类型
				parts = append(parts, fmt.Sprintf("user:%d", userBase.GetID()))
			}
		}
	}

	// 如果按查询参数区分，添加查询字符串
	if options.VaryByQuery && c.Request.URL.RawQuery != "" {
		// 对查询字符串进行哈希以避免键过长
		hash := sha256.Sum256([]byte(c.Request.URL.RawQuery))
		parts = append(parts, "query:"+hex.EncodeToString(hash[:])[:16])
	}

	key := strings.Join(parts, ":")
	return cm.prefix + key
}

// InvalidateCache 使缓存失效
func (cm *CacheMiddleware) InvalidateCache(pattern string) error {
	ctx := context.Background()
	keys, err := cm.client.Keys(ctx, cm.prefix+pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return cm.client.Del(ctx, keys...).Err()
	}

	return nil
}

// InvalidateUserCache 使用户相关的缓存失效
func (cm *CacheMiddleware) InvalidateUserCache(userID uint) error {
	pattern := fmt.Sprintf("*:user:%d*", userID)
	return cm.InvalidateCache(pattern)
}

// InvalidateContentCache 使内容相关的缓存失效（GET /api/content/:id 的 key 含 path /api/content/<id>）
func (cm *CacheMiddleware) InvalidateContentCache(contentID uint) error {
	pattern := fmt.Sprintf("*content/%d*", contentID)
	return cm.InvalidateCache(pattern)
}

// InvalidatePathCache 使特定路径的缓存失效
func (cm *CacheMiddleware) InvalidatePathCache(path string) error {
	pattern := fmt.Sprintf("*:%s*", path)
	return cm.InvalidateCache(pattern)
}

// responseWriter 自定义响应写入器，用于捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *responseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// CacheResponseWriter 用于直接写入缓存响应
type CacheResponseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func NewCacheResponseWriter(w gin.ResponseWriter) *CacheResponseWriter {
	return &CacheResponseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
	}
}

func (w *CacheResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *CacheResponseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (w *CacheResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *CacheResponseWriter) Status() int {
	return w.status
}

func (w *CacheResponseWriter) Body() []byte {
	return w.body.Bytes()
}

// ReadBody 读取响应体（用于测试）
func ReadBody(reader io.Reader) ([]byte, error) {
	return io.ReadAll(reader)
}

// GlobalCacheMiddleware 全局缓存中间件实例（在 main.go 中初始化）
var GlobalCacheMiddleware *CacheMiddleware
