package config

import (
	"log"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// InitSentry 初始化 Sentry 错误监控
func InitSentry() {
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
}

// GetSentryDSN 获取 Sentry DSN
func GetSentryDSN() string {
	return getEnv("SENTRY_DSN", "")
}
