package service

import (
	"backend/entity"
	"backend/repository"
	"encoding/json"
	"fmt"
	"math"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
)

// PurchaseService 购买服务接口
type PurchaseService interface {
	// CreateMembershipCheckout 创建会员购买结账会话（Stripe）
	CreateMembershipCheckout(userID uint, months int) (*stripe.CheckoutSession, error)

	// CreateMembershipCheckoutPayPal 创建会员购买 PayPal 订单（备用支付）
	CreateMembershipCheckoutPayPal(userID uint, months int) (orderID string, err error)

	// CreateCreditsCheckout 创建积分购买结账会话
	CreateCreditsCheckout(userID uint, amount float64, credits int64) (*stripe.CheckoutSession, error)

	// CreateCreditsCheckoutPayPal 创建积分购买 PayPal 订单（备用支付）
	CreateCreditsCheckoutPayPal(userID uint, amount float64, credits int64) (orderID string, err error)

	// PurchaseContentWithCredits 使用积分购买内容
	PurchaseContentWithCredits(userID uint, contentID uint) error

	// HandleCheckoutCompleted 处理结账完成事件（Stripe）
	HandleCheckoutCompleted(sessionID string) error

	// HandlePayPalCapture 处理 PayPal 订单捕获完成（需验证 order 属于 userID）
	HandlePayPalCapture(orderID string, userID uint) error

	// GetPurchaseHistory 获取购买历史
	GetPurchaseHistory(userID uint, limit, offset int) ([]entity.PurchaseBase, error)
}

// purchaseService 购买服务实现
type purchaseService struct {
	stripeService StripeService
	paypalService PayPalService
	purchaseRepo  repository.PurchaseRepository
	contentRepo   repository.ContentRepository
	walletRepo    repository.WalletRepository
	analyticsRepo repository.AnalyticsRepository
}

// NewPurchaseService 创建购买服务实例（paypalService 可选，为 nil 时 PayPal 相关方法返回错误）
func NewPurchaseService(
	stripeService StripeService,
	paypalService PayPalService,
	purchaseRepo repository.PurchaseRepository,
	contentRepo repository.ContentRepository,
	walletRepo repository.WalletRepository,
	analyticsRepo repository.AnalyticsRepository,
) PurchaseService {
	return &purchaseService{
		stripeService: stripeService,
		paypalService: paypalService,
		purchaseRepo:  purchaseRepo,
		contentRepo:   contentRepo,
		walletRepo:    walletRepo,
		analyticsRepo: analyticsRepo,
	}
}

// CreateMembershipCheckout 创建会员购买结账会话
func (s *purchaseService) CreateMembershipCheckout(userID uint, months int) (*stripe.CheckoutSession, error) {
	// 验证月份数
	if months != 1 && months != 3 && months != 6 {
		return nil, fmt.Errorf("invalid months: must be 1, 3, or 6")
	}

	// 创建 Stripe Checkout Session
	checkoutSession, err := s.stripeService.CreateMembershipCheckoutSession(userID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
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

	// 创建待处理的购买记录
	purchase := &entity.PurchaseBase{
		UserID:         userID,
		PurchaseType:   entity.PurchaseTypeMembership,
		Status:         entity.PurchaseStatusPending,
		Amount:         totalAmount,
		Currency:       "USD",
		PaymentGateway: "stripe",
		GatewayOrderID: checkoutSession.ID,
		MembershipType: fmt.Sprintf("%d_months", months),
		ExpiresAt:      nil, // 将在支付完成后设置
	}

	// 序列化 Stripe Session 为 JSON
	sessionJSON, err := json.Marshal(checkoutSession)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}
	purchase.GatewayResponse = string(sessionJSON)

	// 保存购买记录到分片表
	if err := s.purchaseRepo.CreateInShard(userID, purchase); err != nil {
		return nil, fmt.Errorf("failed to create purchase record: %w", err)
	}

	return checkoutSession, nil
}

// membershipAmountAndDesc 返回会员套餐金额（美元字符串）与描述
func membershipAmountAndDesc(months int) (amountUSD string, totalAmount float64, desc string) {
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
	return fmt.Sprintf("%.2f", totalAmount), totalAmount, fmt.Sprintf("Premium membership for %d month(s)", months)
}

// CreateMembershipCheckoutPayPal 创建会员购买 PayPal 订单
func (s *purchaseService) CreateMembershipCheckoutPayPal(userID uint, months int) (orderID string, err error) {
	if months != 1 && months != 3 && months != 6 {
		return "", fmt.Errorf("invalid months: must be 1, 3, or 6")
	}
	if s.paypalService == nil {
		return "", fmt.Errorf("PayPal is not configured")
	}

	amountUSD, totalAmount, desc := membershipAmountAndDesc(months)
	orderID, err = s.paypalService.CreateOrder(amountUSD, "USD", desc)
	if err != nil {
		return "", fmt.Errorf("failed to create PayPal order: %w", err)
	}

	purchase := &entity.PurchaseBase{
		UserID:         userID,
		PurchaseType:   entity.PurchaseTypeMembership,
		Status:         entity.PurchaseStatusPending,
		Amount:         totalAmount,
		Currency:       "USD",
		PaymentGateway: "paypal",
		GatewayOrderID: orderID,
		MembershipType: fmt.Sprintf("%d_months", months),
		ExpiresAt:      nil,
	}
	if err := s.purchaseRepo.CreateInShard(userID, purchase); err != nil {
		return "", fmt.Errorf("failed to create purchase record: %w", err)
	}
	return orderID, nil
}

// CreateCreditsCheckout 创建积分购买结账会话
func (s *purchaseService) CreateCreditsCheckout(userID uint, amount float64, credits int64) (*stripe.CheckoutSession, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	if credits <= 0 {
		return nil, fmt.Errorf("credits must be greater than 0")
	}

	// 创建 Stripe Checkout Session
	checkoutSession, err := s.stripeService.CreateCreditsCheckoutSession(userID, amount, credits)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	// 创建待处理的购买记录
	purchase := &entity.PurchaseBase{
		UserID:         userID,
		PurchaseType:   entity.PurchaseTypeCredits,
		Status:         entity.PurchaseStatusPending,
		Amount:         amount,
		Currency:       "USD",
		PaymentGateway: "stripe",
		GatewayOrderID: checkoutSession.ID,
		Credits:        credits, // 购买的积分数量
	}

	// 序列化 Stripe Session 为 JSON
	sessionJSON, err := json.Marshal(checkoutSession)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}
	purchase.GatewayResponse = string(sessionJSON)

	// 保存购买记录到分片表
	if err := s.purchaseRepo.CreateInShard(userID, purchase); err != nil {
		return nil, fmt.Errorf("failed to create purchase record: %w", err)
	}

	return checkoutSession, nil
}

// CreateCreditsCheckoutPayPal 创建积分购买 PayPal 订单
func (s *purchaseService) CreateCreditsCheckoutPayPal(userID uint, amount float64, credits int64) (orderID string, err error) {
	if amount <= 0 {
		return "", fmt.Errorf("amount must be greater than 0")
	}
	if credits <= 0 {
		return "", fmt.Errorf("credits must be greater than 0")
	}
	if s.paypalService == nil {
		return "", fmt.Errorf("PayPal is not configured")
	}

	amountUSD := fmt.Sprintf("%.2f", amount)
	desc := fmt.Sprintf("Credits - %d credits", credits)
	orderID, err = s.paypalService.CreateOrder(amountUSD, "USD", desc)
	if err != nil {
		return "", fmt.Errorf("failed to create PayPal order: %w", err)
	}

	purchase := &entity.PurchaseBase{
		UserID:         userID,
		PurchaseType:   entity.PurchaseTypeCredits,
		Status:         entity.PurchaseStatusPending,
		Amount:         amount,
		Currency:       "USD",
		PaymentGateway: "paypal",
		GatewayOrderID: orderID,
		Credits:        credits,
	}
	if err := s.purchaseRepo.CreateInShard(userID, purchase); err != nil {
		return "", fmt.Errorf("failed to create purchase record: %w", err)
	}
	return orderID, nil
}

// PurchaseContentWithCredits 使用积分购买内容
func (s *purchaseService) PurchaseContentWithCredits(userID uint, contentID uint) error {
	// 获取内容
	content, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}

	// 检查内容是否有价格（价格即所需积分数，1 积分 = 1）
	if content.Price <= 0 {
		return fmt.Errorf("content is not for sale (price is 0 or not set)")
	}

	// 内容价格即为所需积分数，用户只能用积分购买
	creditsRequired := int64(math.Round(content.Price))
	if creditsRequired <= 0 {
		return fmt.Errorf("content price must be at least 1 credit")
	}

	// 检查用户是否有足够的积分
	hasEnough, err := s.walletRepo.HasEnoughCredits(userID, creditsRequired)
	if err != nil {
		return fmt.Errorf("failed to check credits: %w", err)
	}
	if !hasEnough {
		return fmt.Errorf("insufficient credits: required %d, but user does not have enough", creditsRequired)
	}

	// 扣除积分
	if err := s.walletRepo.SpendCredits(userID, creditsRequired); err != nil {
		return fmt.Errorf("failed to spend credits: %w", err)
	}

	// 创建购买记录
	purchase := &entity.PurchaseBase{
		UserID:         userID,
		PurchaseType:   entity.PurchaseTypeContent,
		Status:         entity.PurchaseStatusCompleted,
		Credits:        creditsRequired, // 扣除的积分数
		Amount:         0,               // 内容购买用积分，无美元金额
		ContentID:      &contentID,
		PaymentGateway: "wallet",
		PaymentMethod:  "wallet",
	}

	// 保存购买记录到分片表
	if err := s.purchaseRepo.CreateInShard(userID, purchase); err != nil {
		// 如果保存失败，尝试回退积分（虽然不太可能发生）
		_ = s.walletRepo.AddCredits(userID, creditsRequired)
		return fmt.Errorf("failed to create purchase record: %w", err)
	}

	// 添加内容销售分析
	if err := s.analyticsRepo.IncrementPurchaseCount(contentID); err != nil {
		// 记录错误但不影响购买流程
		fmt.Printf("Warning: Failed to increment purchase count for content %d: %v\n", contentID, err)
	}

	// 添加收入分析（使用美元价格）
	if err := s.analyticsRepo.AddRevenue(contentID, content.Price); err != nil {
		// 记录错误但不影响购买流程
		fmt.Printf("Warning: Failed to add revenue for content %d: %v\n", contentID, err)
	}

	return nil
}

// HandlePayPalCapture 处理 PayPal 订单捕获完成
func (s *purchaseService) HandlePayPalCapture(orderID string, userID uint) error {
	if s.paypalService == nil {
		return fmt.Errorf("PayPal is not configured")
	}
	_, err := s.paypalService.CaptureOrder(orderID)
	if err != nil {
		return fmt.Errorf("failed to capture PayPal order: %w", err)
	}

	purchase, err := s.purchaseRepo.GetByGatewayOrderID(orderID)
	if err != nil {
		return fmt.Errorf("purchase record not found: %w", err)
	}
	if purchase.UserID != userID {
		return fmt.Errorf("order does not belong to current user")
	}
	if purchase.Status == entity.PurchaseStatusCompleted {
		return nil
	}

	purchase.Status = entity.PurchaseStatusCompleted
	purchase.PaymentMethod = "paypal"

	switch purchase.PurchaseType {
	case entity.PurchaseTypeMembership:
		var months int
		_, _ = fmt.Sscanf(purchase.MembershipType, "%d_months", &months)
		if months <= 0 {
			months = 1
		}
		expiresAt := CalculateMembershipExpiry(months)
		purchase.ExpiresAt = &expiresAt
	case entity.PurchaseTypeCredits:
		if err := s.walletRepo.AddCredits(purchase.UserID, purchase.Credits); err != nil {
			return fmt.Errorf("failed to add credits to wallet: %w", err)
		}
	default:
		return fmt.Errorf("unsupported purchase type for PayPal capture: %s", purchase.PurchaseType)
	}

	if err := s.purchaseRepo.UpdateInShard(purchase.UserID, purchase); err != nil {
		return fmt.Errorf("failed to update purchase record: %w", err)
	}
	return nil
}

// HandleCheckoutCompleted 处理结账完成事件
func (s *purchaseService) HandleCheckoutCompleted(sessionID string) error {
	// 从 Stripe 获取 Checkout Session
	checkoutSession, err := session.Get(sessionID, nil)
	if err != nil {
		return fmt.Errorf("failed to get checkout session: %w", err)
	}

	// 检查支付状态
	if checkoutSession.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return fmt.Errorf("payment not completed, status: %s", checkoutSession.PaymentStatus)
	}

	// 解析元数据
	metadata, err := ParseCheckoutSessionMetadata(checkoutSession)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// 获取用户ID
	userID, err := GetUserIDFromMetadata(metadata)
	if err != nil {
		return fmt.Errorf("failed to get user_id: %w", err)
	}

	// 根据购买类型处理
	purchaseType, err := GetPurchaseTypeFromMetadata(metadata)
	if err != nil {
		return fmt.Errorf("failed to get purchase_type: %w", err)
	}

	// 查找购买记录
	purchase, err := s.purchaseRepo.GetByGatewayOrderID(sessionID)
	if err != nil {
		return fmt.Errorf("purchase record not found: %w", err)
	}

	// 如果已经完成，直接返回
	if purchase.Status == entity.PurchaseStatusCompleted {
		return nil
	}

	// 更新购买记录
	purchase.Status = entity.PurchaseStatusCompleted
	purchase.TransactionID = checkoutSession.PaymentIntent.ID
	if checkoutSession.PaymentMethodTypes != nil && len(checkoutSession.PaymentMethodTypes) > 0 {
		purchase.PaymentMethod = string(checkoutSession.PaymentMethodTypes[0])
	}

	// 更新 GatewayResponse
	sessionJSON, err := json.Marshal(checkoutSession)
	if err == nil {
		purchase.GatewayResponse = string(sessionJSON)
	}

	switch purchaseType {
	case entity.PurchaseTypeMembership:
		// 处理会员购买
		months, err := GetMembershipMonthsFromMetadata(metadata)
		if err != nil {
			return fmt.Errorf("failed to get months: %w", err)
		}

		// 设置会员到期时间
		expiresAt := CalculateMembershipExpiry(months)
		purchase.ExpiresAt = &expiresAt

		// 更新购买记录
		if err := s.purchaseRepo.UpdateInShard(userID, purchase); err != nil {
			return fmt.Errorf("failed to update purchase record: %w", err)
		}

	case entity.PurchaseTypeCredits:
		// 处理积分购买
		credits, err := GetCreditsFromMetadata(metadata)
		if err != nil {
			return fmt.Errorf("failed to get credits: %w", err)
		}

		// 将积分添加到用户钱包
		if err := s.walletRepo.AddCredits(userID, credits); err != nil {
			return fmt.Errorf("failed to add credits to wallet: %w", err)
		}

		// 更新购买记录中的积分数量
		purchase.Credits = credits

		// 更新购买记录
		if err := s.purchaseRepo.UpdateInShard(userID, purchase); err != nil {
			// 如果更新失败，尝试回退积分（虽然不太可能发生）
			_ = s.walletRepo.SpendCredits(userID, credits)
			return fmt.Errorf("failed to update purchase record: %w", err)
		}

	default:
		return fmt.Errorf("unknown purchase type: %s", purchaseType)
	}

	return nil
}

// GetPurchaseHistory 获取购买历史
func (s *purchaseService) GetPurchaseHistory(userID uint, limit, offset int) ([]entity.PurchaseBase, error) {
	return s.purchaseRepo.GetByUserID(userID, limit, offset)
}
