package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter Rate limiter interface
type RateLimiter interface {
	Allow(key string) bool
	Reset(key string)
}

// InMemoryRateLimiter In-memory rate limiter implementation
type InMemoryRateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int           // Number of requests allowed
	window   time.Duration // Time window
}

// Visitor Visitor information
type Visitor struct {
	Count    int
	LastSeen time.Time
}

// NewInMemoryRateLimiter Create new in-memory rate limiter
func NewInMemoryRateLimiter(rate int, window time.Duration) *InMemoryRateLimiter {
	rl := &InMemoryRateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup old visitors periodically
	go rl.cleanup()

	return rl
}

// Allow Check if request is allowed
func (rl *InMemoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	visitor, exists := rl.visitors[key]
	now := time.Now()

	if !exists {
		rl.visitors[key] = &Visitor{
			Count:    1,
			LastSeen: now,
		}
		return true
	}

	// Reset if window has passed
	if now.Sub(visitor.LastSeen) > rl.window {
		visitor.Count = 1
		visitor.LastSeen = now
		return true
	}

	// Check if rate limit exceeded
	if visitor.Count >= rl.rate {
		return false
	}

	visitor.Count++
	visitor.LastSeen = now
	return true
}

// Reset Reset rate limit for a key
func (rl *InMemoryRateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.visitors, key)
}

// cleanup Clean up old visitors periodically
func (rl *InMemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, visitor := range rl.visitors {
			if now.Sub(visitor.LastSeen) > rl.window*2 {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware Rate limiting middleware
func RateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as the key
		key := c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByKey Rate limiting middleware with custom key function
func RateLimitByKey(limiter RateLimiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RedisRateLimiter Redis-based rate limiter implementation
type RedisRateLimiter struct {
	client *redis.Client
	rate   int           // Number of requests allowed
	window time.Duration // Time window
	prefix string        // Redis key prefix
}

// NewRedisRateLimiter Create new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client, rate int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		rate:   rate,
		window: window,
		prefix: "rate_limit:",
	}
}

// Allow Check if request is allowed using Redis
// Uses a sliding window algorithm with Redis INCR and EXPIRE
func (rl *RedisRateLimiter) Allow(key string) bool {
	ctx := context.Background()
	redisKey := rl.prefix + key

	// Use pipeline for atomic operations
	pipe := rl.client.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	// Only set expiration if key is new (TTL returns -1 for keys without expiration)
	ttl := pipe.TTL(ctx, redisKey)
	_, err := pipe.Exec(ctx)

	if err != nil {
		// If Redis fails, allow the request (fail open)
		// In production, you might want to log this error
		return true
	}

	count := incr.Val()
	ttlVal := ttl.Val()

	// If this is a new key (TTL is -1 or -2), set expiration
	if ttlVal < 0 {
		rl.client.Expire(ctx, redisKey, rl.window)
	}

	// Check if rate limit exceeded
	if count > int64(rl.rate) {
		return false
	}

	return true
}

// Reset Reset rate limit for a key
func (rl *RedisRateLimiter) Reset(key string) {
	ctx := context.Background()
	redisKey := rl.prefix + key
	rl.client.Del(ctx, redisKey)
}

// SetPrefix Set Redis key prefix
func (rl *RedisRateLimiter) SetPrefix(prefix string) {
	rl.prefix = prefix
}

// Default rate limiters (will be initialized in main.go with Redis)
var (
	// DefaultRateLimiter Default rate limiter (100 requests per minute)
	// Will be initialized with Redis in main.go
	DefaultRateLimiter RateLimiter

	// StrictRateLimiter Strict rate limiter (10 requests per minute)
	// Will be initialized with Redis in main.go
	StrictRateLimiter RateLimiter

	// AuthRateLimiter Authentication rate limiter (20 requests per minute)
	// Will be initialized with Redis in main.go
	AuthRateLimiter RateLimiter

	// RefreshRateLimiter Refresh token rate limiter (60 requests per minute)
	// More lenient than AuthRateLimiter so multiple tabs/retries can refresh without 429
	RefreshRateLimiter RateLimiter
)
