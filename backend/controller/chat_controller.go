package controller

import (
	"errors"
	"net/http"
	"strings"

	"backend/entity"
	"backend/service"

	"github.com/gin-gonic/gin"
)

// ChatController 聊天：首次需填写名字与人格（存 Postgres），消息历史存 MongoDB
type ChatController struct {
	chatService *service.ChatService
}

// NewChatController 创建聊天控制器
func NewChatController(chatService *service.ChatService) *ChatController {
	return &ChatController{chatService: chatService}
}

// GetSession 获取当前会话状态；若需先填写人格则返回 need_onboarding
// GET /api/chat/session?session_id=xxx
func (c *ChatController) GetSession(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid := userID.(uint)
	sessionID := ctx.Query("session_id")

	need, err := c.chatService.NeedOnboarding(uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if need {
		ctx.JSON(http.StatusOK, gin.H{
			"need_onboarding": true,
			"session_id":      sessionID,
		})
		return
	}

	persona, _ := c.chatService.GetPersona(uid)
	sessCtx := ctx.Request.Context()
	sid, sess, err := c.chatService.GetOrCreateSession(sessCtx, sessionID, uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{
		"need_onboarding": false,
		"session_id":      sid,
		"persona":         persona,
	}
	if sess != nil {
		resp["messages"] = sess.Messages
	}
	ctx.JSON(http.StatusOK, resp)
}

// OnboardingRequest 首次聊天提交的名字与人格
type OnboardingRequest struct {
	Name       string                 `json:"name" binding:"required"`
	Personality entity.PersonalitySchema `json:"personality" binding:"required"`
}

// Onboarding 提交名字与人格，写入 Postgres
// POST /api/chat/onboarding
func (c *ChatController) Onboarding(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid := userID.(uint)

	var req OnboardingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "name and personality required"})
		return
	}
	if err := c.chatService.SavePersona(uid, req.Name, req.Personality); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// StreamRequest 发送消息请求
type StreamRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message" binding:"required"`
}

// Stream 流式回复（SSE）；若未填写人格则 428 Need Persona
// POST /api/chat/stream
func (c *ChatController) Stream(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid := userID.(uint)

	var req StreamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message required"})
		return
	}

	persona, err := c.chatService.GetPersona(uid)
	if err != nil || persona == nil {
		ctx.JSON(http.StatusPreconditionRequired, gin.H{
			"code":    "NEED_PERSONA",
			"message": "Please complete onboarding (name and personality) first",
		})
		return
	}

	// 1 credit per message: deduct before starting SSE so we can return 402
	if err := c.chatService.EnsureAndSpendCreditForMessage(uid); err != nil {
		if errors.Is(err, service.ErrInsufficientCredits) {
			ctx.JSON(http.StatusPaymentRequired, gin.H{
				"code":    "INSUFFICIENT_CREDITS",
				"message": "1 credit per message. Please add credits to continue.",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	sessCtx := ctx.Request.Context()
	_, err = c.chatService.StreamChat(sessCtx, req.SessionID, uid, req.Message, persona, func(text string) {
		_, _ = ctx.Writer.Write([]byte("data: " + escapeSSE(text) + "\n\n"))
		flusher.Flush()
	})
	if err != nil {
		_, _ = ctx.Writer.Write([]byte("data: [ERROR] " + escapeSSE(err.Error()) + "\n\n"))
		flusher.Flush()
		return
	}
}

func escapeSSE(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}
