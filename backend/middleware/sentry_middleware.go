package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
)

// SentryUserMiddleware Sentry用户信息中间件
// 用于在请求中设置用户信息到 Sentry
// 注意：此中间件应该在 sentrygin.New() 之后使用，以便能够访问 Sentry hub
func SentryUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Sentry hub（由 sentrygin 中间件设置）
		if hub := sentrygin.GetHubFromContext(c); hub != nil {
			// 设置用户信息（如果已认证）
			if user, exists := GetUserFromContext(c); exists {
				hub.Scope().SetUser(sentry.User{
					ID:    fmt.Sprintf("%d", user.ID),
					Email: user.Email,
				})
			}

			// 记录请求开始时间
			start := time.Now()

			// 处理请求
			c.Next()

			// 记录请求处理时间
			duration := time.Since(start)

			// 添加性能监控数据
			hub.Scope().SetTag("status_code", fmt.Sprintf("%d", c.Writer.Status()))
			hub.Scope().SetTag("method", c.Request.Method)
			hub.Scope().SetTag("path", c.Request.URL.Path)
			hub.Scope().SetContext("performance", map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			})

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
		} else {
			// 如果没有 hub，继续处理请求（Sentry 可能未配置）
			c.Next()
		}
	}
}

// GetSentryHub 从上下文中获取 Sentry hub（使用官方 SDK 的方法）
func GetSentryHub(c *gin.Context) (*sentry.Hub, bool) {
	hub := sentrygin.GetHubFromContext(c)
	return hub, hub != nil
}
