package controller

import (
	"backend/entity"
	"backend/middleware"
	"backend/service"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/webhook"
)

// PurchaseController 购买控制器
type PurchaseController struct {
	purchaseService service.PurchaseService
	stripeService   service.StripeService
}

// NewPurchaseController 创建购买控制器实例
func NewPurchaseController(
	purchaseService service.PurchaseService,
	stripeService service.StripeService,
) *PurchaseController {
	return &PurchaseController{
		purchaseService: purchaseService,
		stripeService:   stripeService,
	}
}

// CreateMembershipCheckoutRequest 创建会员购买结账请求
type CreateMembershipCheckoutRequest struct {
	Months int `json:"months" binding:"required,oneof=1 3 6"` // 1, 3, 或 6 个月
}

// CreateMembershipCheckout 创建会员购买结账会话
// @Summary Create membership checkout session
// @Description Create a Stripe checkout session for membership purchase
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateMembershipCheckoutRequest true "Membership checkout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/purchase/membership/checkout [post]
func (pc *PurchaseController) CreateMembershipCheckout(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 解析请求
	var req CreateMembershipCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 验证月份数
	if req.Months != 1 && req.Months != 3 && req.Months != 9 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "months must be 1, 3, or 6",
		})
		return
	}

	// 创建结账会话
	checkoutSession, err := pc.purchaseService.CreateMembershipCheckout(userBase.ID, req.Months)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create checkout session",
			"details": err.Error(),
		})
		return
	}

	// 返回格式匹配 Stripe 文档示例
	c.JSON(http.StatusOK, gin.H{
		"clientSecret": checkoutSession.ClientSecret, // 匹配文档格式（驼峰命名）
		// 额外信息（可选）
		"checkout_session_id": checkoutSession.ID,
		"url":                 checkoutSession.URL,
	})
}

// CreatePayPalMembershipOrder 创建会员购买 PayPal 订单（备用支付）
// @Summary Create PayPal order for membership
// @Description Create a PayPal order for membership purchase (backup payment)
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateMembershipCheckoutRequest true "Months: 1, 3, or 6"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /api/purchase/membership/paypal/order [post]
func (pc *PurchaseController) CreatePayPalMembershipOrder(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userBase := user.(*entity.UserBase)

	var req CreateMembershipCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}
	if req.Months != 1 && req.Months != 3 && req.Months != 9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "months must be 1, 3, or 6"})
		return
	}

	orderID, err := pc.purchaseService.CreateMembershipCheckoutPayPal(userBase.ID, req.Months)
	if err != nil {
		if err.Error() == "PayPal is not configured" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PayPal is not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PayPal order", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": orderID})
}

// CapturePayPalOrderRequest 捕获 PayPal 订单请求
type CapturePayPalOrderRequest struct {
	OrderID string `json:"orderID" binding:"required"`
}

// CapturePayPalOrder 捕获 PayPal 订单完成支付
// @Summary Capture PayPal order
// @Description Capture the PayPal order to complete payment and grant membership
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CapturePayPalOrderRequest true "PayPal order ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /api/purchase/membership/paypal/capture [post]
func (pc *PurchaseController) CapturePayPalOrder(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userBase := user.(*entity.UserBase)

	var req CapturePayPalOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	if err := pc.purchaseService.HandlePayPalCapture(req.OrderID, userBase.ID); err != nil {
		if err.Error() == "PayPal is not configured" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PayPal is not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to capture PayPal order", "details": err.Error()})
		return
	}
	if middleware.GlobalCacheMiddleware != nil {
		userBase := user.(*entity.UserBase)
		_ = middleware.GlobalCacheMiddleware.InvalidateUserCache(userBase.ID)
	}
	c.JSON(http.StatusOK, gin.H{"status": "completed"})
}

// CreateCreditsCheckoutRequest 创建积分购买结账请求
type CreateCreditsCheckoutRequest struct {
	Amount  float64 `json:"amount" binding:"required,gt=0"`  // 支付金额（美元）
	Credits int64   `json:"credits" binding:"required,gt=0"` // 购买的积分数量
}

// CreatePayPalCreditsOrder 创建积分购买 PayPal 订单（备用支付）
// @Summary Create PayPal order for credits
// @Description Create a PayPal order for credits purchase (backup payment)
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateCreditsCheckoutRequest true "Amount and credits"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /api/purchase/credits/paypal/order [post]
func (pc *PurchaseController) CreatePayPalCreditsOrder(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userBase := user.(*entity.UserBase)

	var req CreateCreditsCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	orderID, err := pc.purchaseService.CreateCreditsCheckoutPayPal(userBase.ID, req.Amount, req.Credits)
	if err != nil {
		if err.Error() == "PayPal is not configured" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "PayPal is not configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PayPal order", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": orderID})
}

// CreateCreditsCheckout 创建积分购买结账会话
// @Summary Create credits checkout session
// @Description Create a Stripe checkout session for credits purchase
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateCreditsCheckoutRequest true "Credits checkout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/purchase/credits/checkout [post]
func (pc *PurchaseController) CreateCreditsCheckout(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 解析请求
	var req CreateCreditsCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 创建结账会话
	checkoutSession, err := pc.purchaseService.CreateCreditsCheckout(userBase.ID, req.Amount, req.Credits)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create checkout session",
			"details": err.Error(),
		})
		return
	}

	// 返回格式匹配 Stripe 文档示例
	c.JSON(http.StatusOK, gin.H{
		"clientSecret": checkoutSession.ClientSecret, // 匹配文档格式（驼峰命名）
		// 额外信息（可选）
		"checkout_session_id": checkoutSession.ID,
		"url":                 checkoutSession.URL,
	})
}

// PurchaseContentRequest 购买内容请求
type PurchaseContentRequest struct {
	ContentID uint `json:"content_id" binding:"required"`
}

// PurchaseContent 使用积分购买内容
// @Summary Purchase content with credits
// @Description Purchase content using wallet credits
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body PurchaseContentRequest true "Purchase content request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/purchase/content [post]
func (pc *PurchaseController) PurchaseContent(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 解析请求
	var req PurchaseContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 使用积分购买内容
	if err := pc.purchaseService.PurchaseContentWithCredits(userBase.ID, req.ContentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to purchase content",
			"details": err.Error(),
		})
		return
	}

	// 使内容缓存和用户购买历史缓存失效
	if middleware.GlobalCacheMiddleware != nil {
		middleware.GlobalCacheMiddleware.InvalidateContentCache(req.ContentID)
		middleware.GlobalCacheMiddleware.InvalidateUserCache(userBase.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Content purchased successfully",
	})
}

// VerifyCheckoutRequest 验证结账请求
type VerifyCheckoutRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// VerifyCheckout 验证结账状态
// @Summary Verify checkout status
// @Description Verify the status of a checkout session (similar to retrieveCheckoutSession in Stripe docs)
// @Tags purchase
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body VerifyCheckoutRequest true "Verify checkout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/purchase/verify [post]
func (pc *PurchaseController) VerifyCheckout(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 解析请求
	var req VerifyCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 获取 Checkout Session（使用 AddExpand 获取 payment_intent 信息，匹配文档示例）
	params := &stripe.CheckoutSessionParams{}
	params.AddExpand("payment_intent")
	session, err := session.Get(req.SessionID, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to retrieve checkout session",
			"details": err.Error(),
		})
		return
	}

	// 如果支付已完成，处理购买完成逻辑
	if session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
		// 处理结账完成（添加积分或会员等）
		if err := pc.purchaseService.HandleCheckoutCompleted(req.SessionID); err != nil {
			// 记录错误但不影响状态返回
			fmt.Printf("Warning: Failed to handle checkout completed: %v\n", err)
		}
		// 使该用户的 /api/auth/me 缓存失效，以便前端立即看到更新后的钱包余额
		if middleware.GlobalCacheMiddleware != nil {
			_ = middleware.GlobalCacheMiddleware.InvalidateUserCache(userBase.ID)
		}
	}

	// 返回状态信息（匹配文档格式）
	response := gin.H{
		"status":         string(session.Status),
		"payment_status": string(session.PaymentStatus),
	}

	// 如果有 payment_intent，添加相关信息
	if session.PaymentIntent != nil {
		response["payment_intent_id"] = session.PaymentIntent.ID
		response["payment_intent_status"] = string(session.PaymentIntent.Status)
	}

	c.JSON(http.StatusOK, response)
}

// GetPurchaseHistory 获取购买历史
// @Summary Get purchase history
// @Description Get purchase history for the current user
// @Tags purchase
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/purchase/history [get]
func (pc *PurchaseController) GetPurchaseHistory(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 获取查询参数
	limit := 20
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
			limit = 20
		}
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &offset); err != nil {
			offset = 0
		}
		if offset < 0 {
			offset = 0
		}
	}

	// 获取购买历史
	purchases, err := pc.purchaseService.GetPurchaseHistory(userBase.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get purchase history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"purchases": purchases,
		"limit":     limit,
		"offset":    offset,
	})
}

// HandleStripeWebhook 处理 Stripe webhook
// @Summary Handle Stripe webhook
// @Description Handle Stripe webhook events (following Stripe's official example)
// @Tags purchase
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/purchase/webhook [post]
func (pc *PurchaseController) HandleStripeWebhook(c *gin.Context) {
	const MaxBodyBytes = int64(65536) // 限制请求体大小为 64KB

	// 限制请求体大小（匹配 Stripe 示例）
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	// 读取请求体
	payload, err := c.GetRawData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	// 先解析基本事件结构（匹配 Stripe 示例）
	event := stripe.Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Webhook error while parsing basic request. %v\n", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to parse webhook event",
			"details": err.Error(),
		})
		return
	}

	// 获取 webhook secret（从环境变量）
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		fmt.Fprintf(os.Stderr, "⚠️  STRIPE_WEBHOOK_SECRET not configured\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "STRIPE_WEBHOOK_SECRET not configured",
		})
		return
	}

	// 获取签名头
	signatureHeader := c.GetHeader("Stripe-Signature")
	if signatureHeader == "" {
		signatureHeader = c.GetHeader("X-Stripe-Signature")
	}

	// 验证 webhook 签名（匹配 Stripe 示例）
	event, err = webhook.ConstructEvent(payload, signatureHeader, webhookSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Webhook signature verification failed. %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid webhook signature",
			"details": err.Error(),
		})
		return
	}

	// 根据事件类型处理（匹配 Stripe 示例模式）
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		// 解析 Checkout Session
		var checkoutSession stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &checkoutSession); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to parse checkout session",
				"details": err.Error(),
			})
			return
		}

		// 处理结账完成
		if err := pc.purchaseService.HandleCheckoutCompleted(checkoutSession.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error handling checkout completed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to handle checkout completed",
				"details": err.Error(),
			})
			return
		}
		// 使该用户的 /api/auth/me 缓存失效，以便前端立即看到更新后的钱包余额
		if middleware.GlobalCacheMiddleware != nil && checkoutSession.Metadata != nil {
			if userIDStr, ok := checkoutSession.Metadata["user_id"]; ok && userIDStr != "" {
				if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
					_ = middleware.GlobalCacheMiddleware.InvalidateUserCache(uint(userID))
				}
			}
		}

	case "payment_intent.succeeded":
		// 处理支付成功事件（匹配 Stripe 示例）
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to parse payment intent",
				"details": err.Error(),
			})
			return
		}
		// 记录支付成功（可以根据需要添加处理逻辑）
		fmt.Printf("Successful payment for %d cents.\n", paymentIntent.Amount)

	case "payment_method.attached":
		// 处理支付方式附加事件（匹配 Stripe 示例）
		var paymentMethod stripe.PaymentMethod
		if err := json.Unmarshal(event.Data.Raw, &paymentMethod); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to parse payment method",
				"details": err.Error(),
			})
			return
		}
		// 记录支付方式附加（可以根据需要添加处理逻辑）
		fmt.Printf("Payment method attached: %s\n", paymentMethod.ID)

	default:
		// 未处理的事件类型（匹配 Stripe 示例）
		fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		c.JSON(http.StatusOK, gin.H{
			"message": "Event received but not processed",
			"type":    event.Type,
		})
		return
	}

	// 返回成功（匹配 Stripe 示例）
	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook processed successfully",
	})
}
