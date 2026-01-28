package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/getsentry/sentry-go"
)

// SentryMiddleware Sentry日志中间件
// 用于捕获和记录请求中的错误和异常
func SentryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建 Sentry hub 用于当前请求
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(c.Request)
		
		// 设置用户信息（如果已认证）
		if user, exists := GetUserFromContext(c); exists {
			hub.Scope().SetUser(sentry.User{
				ID:    fmt.Sprintf("%d", user.ID),
				Email: user.Email,
			})
		}

		// 设置请求上下文
		c.Set("sentry_hub", hub)

		// 记录请求开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录请求处理时间
		duration := time.Since(start)

		// 如果有错误，发送到 Sentry
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				hub.CaptureException(err)
			}
		}

		// 如果状态码 >= 500，记录错误
		if c.Writer.Status() >= 500 {
			hub.Scope().SetLevel(sentry.LevelError)
			hub.CaptureMessage("Server error occurred")
		}

		// 添加性能监控数据
		hub.Scope().SetTag("status_code", fmt.Sprintf("%d", c.Writer.Status()))
		hub.Scope().SetTag("method", c.Request.Method)
		hub.Scope().SetTag("path", c.Request.URL.Path)
		hub.Scope().SetContext("performance", map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
		})
	}
}

// SentryRecoveryMiddleware Sentry恢复中间件
// 用于捕获 panic 并发送到 Sentry
func SentryRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取 Sentry hub
				hub, exists := c.Get("sentry_hub")
				if exists {
					if sentryHub, ok := hub.(*sentry.Hub); ok {
						// 将 panic 值转换为 error
						var errObj error
						if e, ok := err.(error); ok {
							errObj = e
						} else {
							errObj = fmt.Errorf("%v", err)
						}
						sentryHub.CaptureException(errObj)
					}
				} else {
					// 如果没有 hub，使用默认的
					var errObj error
					if e, ok := err.(error); ok {
						errObj = e
					} else {
						errObj = fmt.Errorf("%v", err)
					}
					sentry.CaptureException(errObj)
				}

				// 重新抛出 panic，让 Gin 的默认恢复处理器处理
				panic(err)
			}
		}()

		c.Next()
	}
}

// GetSentryHub 从上下文中获取 Sentry hub
func GetSentryHub(c *gin.Context) (*sentry.Hub, bool) {
	hub, exists := c.Get("sentry_hub")
	if !exists {
		return nil, false
	}

	sentryHub, ok := hub.(*sentry.Hub)
	return sentryHub, ok
}
