package service

import (
	"backend/entity"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
)

// getStripeReturnURL returns a valid absolute URL for Stripe return_url (e.g. checkout session).
// FRONTEND_URL must be set to a valid base URL (e.g. http://localhost:3000 or https://app.example.com).
func getStripeReturnURL(path string) string {
	base := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if base == "" {
		base = "http://localhost:3000"
	}
	base = strings.TrimSuffix(base, "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	path = strings.TrimPrefix(path, "/")
	return base + "/" + path
}

// validateReturnURL ensures the URL is valid for Stripe (absolute with scheme).
func validateReturnURL(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("return_url must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("return_url must have a host")
	}
	return nil
}

// StripeService Stripe 支付服务接口
type StripeService interface {
	// CreateMembershipCheckoutSession 创建会员购买结账会话
	CreateMembershipCheckoutSession(userID uint, months int) (*stripe.CheckoutSession, error)

	// CreateCreditsCheckoutSession 创建积分购买结账会话
	CreateCreditsCheckoutSession(userID uint, amount float64, credits int64) (*stripe.CheckoutSession, error)

	// HandleWebhook 处理 Stripe webhook 事件
	HandleWebhook(event *stripe.Event) error
}

// stripeService Stripe 服务实现
type stripeService struct {
	webhookSecret string
}

// NewStripeService 创建 Stripe 服务实例
func NewStripeService() (StripeService, error) {
	// 从环境变量获取 Stripe API Key
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY environment variable is not set")
	}

	// 设置 Stripe API Key
	stripe.Key = stripeKey

	// 从环境变量获取 Webhook Secret（可选）
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	return &stripeService{
		webhookSecret: webhookSecret,
	}, nil
}

// CreateMembershipCheckoutSession 创建会员购买结账会话
func (s *stripeService) CreateMembershipCheckoutSession(userID uint, months int) (*stripe.CheckoutSession, error) {
	// 验证月份数
	if months != 1 && months != 3 && months != 6 {
		return nil, fmt.Errorf("invalid months: must be 1, 3, or 6")
	}

	// 价格与前端 subscribe 页一致：1 月 6.99，3 月 16.99，6 月 36.99
	var totalAmount float64
	switch months {
	case 1:
		totalAmount = 6.99
	case 3:
		totalAmount = 16.99
	case 6:
		totalAmount = 36.99
	default:
		totalAmount = 6.99 * float64(months)
	}

	returnURL := getStripeReturnURL("purchase/return?session_id={CHECKOUT_SESSION_ID}")
	if err := validateReturnURL(returnURL); err != nil {
		return nil, fmt.Errorf("invalid FRONTEND_URL for Stripe return_url: %w", err)
	}

	// 创建 Checkout Session
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyUSD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(fmt.Sprintf("Membership - %d month(s)", months)),
						Description: stripe.String(fmt.Sprintf("Premium membership for %d month(s) — $%.2f total", months, totalAmount)),
					},
					UnitAmount: stripe.Int64(int64(totalAmount * 100)), // Stripe 使用分为单位
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:      stripe.String(string(stripe.CheckoutSessionModePayment)),
		UIMode:    stripe.String("custom"),
		ReturnURL: stripe.String(returnURL),
		Metadata: map[string]string{
			"user_id":         fmt.Sprintf("%d", userID),
			"purchase_type":   "membership",
			"months":          fmt.Sprintf("%d", months),
			"membership_type": fmt.Sprintf("%d_months", months),
		},
	}

	result, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return result, nil
}

// CreateCreditsCheckoutSession 创建积分购买结账会话
func (s *stripeService) CreateCreditsCheckoutSession(userID uint, amount float64, credits int64) (*stripe.CheckoutSession, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	if credits <= 0 {
		return nil, fmt.Errorf("credits must be greater than 0")
	}

	returnURL := getStripeReturnURL("purchase/return?session_id={CHECKOUT_SESSION_ID}")
	if err := validateReturnURL(returnURL); err != nil {
		return nil, fmt.Errorf("invalid FRONTEND_URL for Stripe return_url: %w", err)
	}

	// 创建 Checkout Session
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(string(stripe.CurrencyUSD)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(fmt.Sprintf("Credits - %d credits", credits)),
						Description: stripe.String(fmt.Sprintf("Purchase %d credits for $%.2f", credits, amount)),
					},
					UnitAmount: stripe.Int64(int64(amount * 100)), // Stripe 使用分为单位
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:      stripe.String(string(stripe.CheckoutSessionModePayment)),
		UIMode:    stripe.String("custom"),
		ReturnURL: stripe.String(returnURL),
		Metadata: map[string]string{
			"user_id":       fmt.Sprintf("%d", userID),
			"purchase_type": "credits",
			"credits":       fmt.Sprintf("%d", credits),
		},
	}

	result, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return result, nil
}

// HandleWebhook 处理 Stripe webhook 事件
func (s *stripeService) HandleWebhook(event *stripe.Event) error {
	// 这里只是返回事件，实际处理逻辑在购买服务中
	// 因为需要访问数据库和仓库
	return nil
}

// ParseCheckoutSessionMetadata 解析 Checkout Session 的元数据
func ParseCheckoutSessionMetadata(session *stripe.CheckoutSession) (map[string]string, error) {
	if session.Metadata == nil {
		return nil, fmt.Errorf("session metadata is nil")
	}
	return session.Metadata, nil
}

// GetPurchaseTypeFromMetadata 从元数据中获取购买类型
func GetPurchaseTypeFromMetadata(metadata map[string]string) (entity.PurchaseType, error) {
	purchaseTypeStr, ok := metadata["purchase_type"]
	if !ok {
		return "", fmt.Errorf("purchase_type not found in metadata")
	}

	switch purchaseTypeStr {
	case "membership":
		return entity.PurchaseTypeMembership, nil
	case "credits":
		return entity.PurchaseTypeCredits, nil
	default:
		return "", fmt.Errorf("unknown purchase_type: %s", purchaseTypeStr)
	}
}

// GetUserIDFromMetadata 从元数据中获取用户ID
func GetUserIDFromMetadata(metadata map[string]string) (uint, error) {
	userIDStr, ok := metadata["user_id"]
	if !ok {
		return 0, fmt.Errorf("user_id not found in metadata")
	}

	var userID uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		return 0, fmt.Errorf("invalid user_id format: %w", err)
	}

	return userID, nil
}

// GetCreditsFromMetadata 从元数据中获取积分数量
func GetCreditsFromMetadata(metadata map[string]string) (int64, error) {
	creditsStr, ok := metadata["credits"]
	if !ok {
		return 0, fmt.Errorf("credits not found in metadata")
	}

	var credits int64
	if _, err := fmt.Sscanf(creditsStr, "%d", &credits); err != nil {
		return 0, fmt.Errorf("invalid credits format: %w", err)
	}

	return credits, nil
}

// GetMembershipMonthsFromMetadata 从元数据中获取会员月数
func GetMembershipMonthsFromMetadata(metadata map[string]string) (int, error) {
	monthsStr, ok := metadata["months"]
	if !ok {
		return 0, fmt.Errorf("months not found in metadata")
	}

	var months int
	if _, err := fmt.Sscanf(monthsStr, "%d", &months); err != nil {
		return 0, fmt.Errorf("invalid months format: %w", err)
	}

	return months, nil
}

// GetMembershipTypeFromMetadata 从元数据中获取会员类型
func GetMembershipTypeFromMetadata(metadata map[string]string) (string, error) {
	membershipType, ok := metadata["membership_type"]
	if !ok {
		return "", fmt.Errorf("membership_type not found in metadata")
	}
	return membershipType, nil
}

// CalculateMembershipExpiry 计算会员到期时间
func CalculateMembershipExpiry(months int) time.Time {
	return time.Now().AddDate(0, months, 0)
}

// ConvertStripeAmountToFloat 将 Stripe 金额（分）转换为浮点数（美元）
func ConvertStripeAmountToFloat(amount int64) float64 {
	return float64(amount) / 100.0
}

// ConvertFloatToStripeAmount 将浮点数（美元）转换为 Stripe 金额（分）
func ConvertFloatToStripeAmount(amount float64) int64 {
	return int64(amount * 100)
}

// SerializeStripeEvent 序列化 Stripe 事件为 JSON
func SerializeStripeEvent(event *stripe.Event) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event: %w", err)
	}
	return string(data), nil
}
