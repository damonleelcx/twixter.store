package entity

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PurchaseType 购买类型枚举
type PurchaseType string

const (
	PurchaseTypeCredits    PurchaseType = "credits"    // 购买积分（从支付网关）
	PurchaseTypeMembership PurchaseType = "membership" // 购买会员（使用积分）
	PurchaseTypeContent    PurchaseType = "content"    // 购买内容（使用积分）
)

// PurchaseStatus 购买状态枚举
type PurchaseStatus string

const (
	PurchaseStatusPending   PurchaseStatus = "pending"   // 待处理
	PurchaseStatusCompleted PurchaseStatus = "completed" // 已完成
	PurchaseStatusFailed    PurchaseStatus = "failed"    // 失败
	PurchaseStatusRefunded  PurchaseStatus = "refunded"  // 已退款
)

// PurchaseBase 购买基础字段（用于分片表的共享字段）
type PurchaseBase struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"not null;index" json:"user_id"`                                   // 购买者用户ID
	PurchaseType PurchaseType   `gorm:"type:varchar(20);not null;index" json:"purchase_type"`            // 购买类型：credits, membership, content
	Status       PurchaseStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"` // 购买状态
	Credits      int64          `gorm:"not null;default:0" json:"credits"`                               // 积分数量（购买积分时表示购买的积分，购买会员/内容时表示使用的积分）

	// 购买积分相关（当 PurchaseType 为 credits 时使用）
	Amount          float64 `gorm:"type:decimal(10,2);default:0" json:"amount,omitempty"` // 支付金额（仅当购买积分时使用）
	Currency        string  `gorm:"size:10;default:'USD'" json:"currency,omitempty"`      // 货币类型（如 USD, CNY）
	PaymentGateway  string  `gorm:"size:50;index" json:"payment_gateway,omitempty"`       // 支付网关名称（如 stripe, paypal, alipay）
	GatewayOrderID  string  `gorm:"size:255;index" json:"gateway_order_id,omitempty"`     // 支付网关订单ID
	GatewayResponse string  `gorm:"type:text" json:"gateway_response,omitempty"`          // 支付网关响应数据（JSON格式）

	// 购买内容相关（当 PurchaseType 为 content 时使用）
	ContentID *uint `gorm:"index" json:"content_id,omitempty"` // 购买的内容ID（可为空）

	// 购买会员相关（当 PurchaseType 为 membership 时使用）
	MembershipType string     `gorm:"size:50;index" json:"membership_type,omitempty"` // 会员类型（可为空）
	ExpiresAt      *time.Time `gorm:"type:timestamp" json:"expires_at,omitempty"`     // 会员到期时间（可为空）

	// 交易信息
	TransactionID string `gorm:"size:255;index" json:"transaction_id,omitempty"` // 交易ID（可选）
	PaymentMethod string `gorm:"size:50" json:"payment_method,omitempty"`        // 支付方式（可选，如 credit_card, debit_card, wallet）

	// 元数据
	Notes string `gorm:"type:text" json:"notes,omitempty"` // 备注信息

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// PurchaseShard0 分片0的购买表
type PurchaseShard0 struct {
	PurchaseBase
}

// TableName 指定表名
func (PurchaseShard0) TableName() string {
	return "purchases_shard_0"
}

// BeforeCreate 创建前的钩子函数
func (p *PurchaseShard0) BeforeCreate(tx *gorm.DB) error {
	// 设置默认状态
	if p.Status == "" {
		p.Status = PurchaseStatusPending
	}
	return nil
}

// PurchaseShard1 分片1的购买表
type PurchaseShard1 struct {
	PurchaseBase
}

// TableName 指定表名
func (PurchaseShard1) TableName() string {
	return "purchases_shard_1"
}

// BeforeCreate 创建前的钩子函数
func (p *PurchaseShard1) BeforeCreate(tx *gorm.DB) error {
	// 设置默认状态
	if p.Status == "" {
		p.Status = PurchaseStatusPending
	}
	return nil
}

// Purchase 通用购买接口（用于向后兼容和辅助函数）
type Purchase struct {
	PurchaseBase
}

// TableName 指定表名（默认表，用于向后兼容）
func (Purchase) TableName() string {
	return "purchases"
}

// BeforeCreate 创建前的钩子函数
func (p *Purchase) BeforeCreate(tx *gorm.DB) error {
	// 设置默认状态
	if p.Status == "" {
		p.Status = PurchaseStatusPending
	}
	return nil
}

// GetPurchaseShardNumber 根据用户ID获取购买表分片编号（用于分片策略）
func GetPurchaseShardNumber(userID uint) int {
	return int(userID % 2)
}

// GetPurchaseShardTableName 根据用户ID获取购买表分片表名
func GetPurchaseShardTableName(userID uint) string {
	shardNum := GetPurchaseShardNumber(userID)
	return fmt.Sprintf("purchases_shard_%d", shardNum)
}

// IsCreditsPurchase 检查是否为购买积分
func (p *PurchaseBase) IsCreditsPurchase() bool {
	return p.PurchaseType == PurchaseTypeCredits
}

// IsMembershipPurchase 检查是否为会员购买
func (p *PurchaseBase) IsMembershipPurchase() bool {
	return p.PurchaseType == PurchaseTypeMembership
}

// IsContentPurchase 检查是否为内容购买
func (p *PurchaseBase) IsContentPurchase() bool {
	return p.PurchaseType == PurchaseTypeContent
}

// IsCompleted 检查购买是否已完成
func (p *PurchaseBase) IsCompleted() bool {
	return p.Status == PurchaseStatusCompleted
}

// IsPending 检查购买是否待处理
func (p *PurchaseBase) IsPending() bool {
	return p.Status == PurchaseStatusPending
}

// IsFailed 检查购买是否失败
func (p *PurchaseBase) IsFailed() bool {
	return p.Status == PurchaseStatusFailed
}

// IsRefunded 检查购买是否已退款
func (p *PurchaseBase) IsRefunded() bool {
	return p.Status == PurchaseStatusRefunded
}
