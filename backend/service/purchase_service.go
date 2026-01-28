package service

import (
	"backend/entity"
	"backend/repository"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
)

// PurchaseService 购买服务接口
type PurchaseService interface {
	// CreateMembershipCheckout 创建会员购买结账会话
	CreateMembershipCheckout(userID uint, months int) (*stripe.CheckoutSession, error)

	// CreateCreditsCheckout 创建积分购买结账会话
	CreateCreditsCheckout(userID uint, amount float64, credits int64) (*stripe.CheckoutSession, error)

	// PurchaseContentWithCredits 使用积分购买内容
	PurchaseContentWithCredits(userID uint, contentID uint) error

	// HandleCheckoutCompleted 处理结账完成事件
	HandleCheckoutCompleted(sessionID string) error

	// GetPurchaseHistory 获取购买历史
	GetPurchaseHistory(userID uint, limit, offset int) ([]entity.PurchaseBase, error)
}

// purchaseService 购买服务实现
type purchaseService struct {
	stripeService StripeService
	purchaseRepo  repository.PurchaseRepository
	contentRepo   repository.ContentRepository
	walletRepo    repository.WalletRepository
	analyticsRepo repository.AnalyticsRepository
}

// NewPurchaseService 创建购买服务实例
func NewPurchaseService(
	stripeService StripeService,
	purchaseRepo repository.PurchaseRepository,
	contentRepo repository.ContentRepository,
	walletRepo repository.WalletRepository,
	analyticsRepo repository.AnalyticsRepository,
) PurchaseService {
	return &purchaseService{
		stripeService: stripeService,
		purchaseRepo:  purchaseRepo,
		contentRepo:   contentRepo,
		walletRepo:    walletRepo,
		analyticsRepo: analyticsRepo,
	}
}

// CreateMembershipCheckout 创建会员购买结账会话
func (s *purchaseService) CreateMembershipCheckout(userID uint, months int) (*stripe.CheckoutSession, error) {
	// 验证月份数
	if months != 1 && months != 3 && months != 9 {
		return nil, fmt.Errorf("invalid months: must be 1, 3, or 9")
	}

	// 创建 Stripe Checkout Session
	checkoutSession, err := s.stripeService.CreateMembershipCheckoutSession(userID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	// 计算价格
	monthlyPrice := 6.99
	totalAmount := monthlyPrice * float64(months)

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

// PurchaseContentWithCredits 使用积分购买内容
func (s *purchaseService) PurchaseContentWithCredits(userID uint, contentID uint) error {
	// 获取内容
	content, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}

	// 检查内容是否有价格
	if content.Price <= 0 {
		return fmt.Errorf("content is not for sale (price is 0 or not set)")
	}

	// 将价格转换为积分（假设 1 美元 = 100 积分，可以根据实际需求调整）
	creditsRequired := int64(content.Price * 100)

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
		Credits:        creditsRequired, // 使用的积分数量
		Amount:         content.Price,   // 内容价格（美元）
		Currency:       "USD",
		ContentID:      &contentID,
		PaymentGateway: "wallet", // 使用钱包支付
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
