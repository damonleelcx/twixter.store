package entity

import (
	"time"

	"gorm.io/gorm"
)

// Wallet 钱包模型（存储用户积分余额）
type Wallet struct {
	ID          uint  `gorm:"primaryKey" json:"id"`
	UserID      uint  `gorm:"uniqueIndex;not null;index" json:"user_id"` // 用户ID（唯一索引，每个用户只有一个钱包）
	ShardNumber int   `gorm:"not null;index" json:"shard_number"`        // 分片编号（0 或 1）
	Balance     int64 `gorm:"not null;default:0" json:"balance"`         // 当前积分余额
	TotalEarned int64 `gorm:"not null;default:0" json:"total_earned"`    // 累计获得的积分（购买积分、奖励等）
	TotalSpent  int64 `gorm:"not null;default:0" json:"total_spent"`     // 累计消费的积分（购买会员、内容等）

	// 锁定相关（用于防止并发问题）
	LockedAt *time.Time `gorm:"type:timestamp" json:"locked_at,omitempty"` // 锁定时间（用于事务处理）

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// TableName 指定表名
func (Wallet) TableName() string {
	return "wallets"
}

// BeforeCreate 创建前的钩子函数
func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	// 自动计算分片编号（如果未设置）
	if w.UserID > 0 {
		w.ShardNumber = GetShardNumber(w.UserID)
	}
	return nil
}

// AddCredits 增加积分（用于购买积分、奖励等）
func (w *Wallet) AddCredits(amount int64) {
	if amount > 0 {
		w.Balance += amount
		w.TotalEarned += amount
	}
}

// SpendCredits 消费积分（用于购买会员、内容等）
func (w *Wallet) SpendCredits(amount int64) bool {
	if amount > 0 && w.Balance >= amount {
		w.Balance -= amount
		w.TotalSpent += amount
		return true
	}
	return false
}

// HasEnoughCredits 检查是否有足够的积分
func (w *Wallet) HasEnoughCredits(amount int64) bool {
	return w.Balance >= amount
}

// GetBalance 获取当前余额
func (w *Wallet) GetBalance() int64 {
	return w.Balance
}
