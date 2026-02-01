package entity

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AccountType 账户类型枚举
type AccountType string

const (
	AccountTypeLight AccountType = "light"
	AccountTypeDark  AccountType = "dark"
	AccountTypeAdmin AccountType = "admin"
)

// UserBase 用户基础字段（用于分片表的共享字段）
type UserBase struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	Email       string      `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Username    string      `gorm:"uniqueIndex;size:100" json:"username"` // 用户名（可选，可为空）
	Password    string      `gorm:"not null;size:255" json:"-"`           // 不序列化密码字段
	AccountType AccountType `gorm:"type:varchar(20);not null;default:'light'" json:"account_type"`
	IPAddress   string      `gorm:"size:45" json:"ip_address"` // IPv6 最大长度为 45

	// 推荐相关字段
	ReferralCode string `gorm:"uniqueIndex;size:20;index" json:"referral_code"` // 用户的推荐码（唯一）
	ReferredBy   *uint  `gorm:"index" json:"referred_by,omitempty"`             // 推荐人ID（可为空）

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除

	// 安全相关字段
	LastLoginAt          *time.Time `gorm:"type:timestamp" json:"last_login_at"`        // 最后登录时间
	FailedLoginAttempts  int        `gorm:"default:0" json:"failed_login_attempts"`     // 失败登录次数
	AccountLocked        bool       `gorm:"default:false" json:"account_locked"`        // 账户是否被锁定
	AccountLockedUntil   *time.Time `gorm:"type:timestamp" json:"account_locked_until"` // 账户锁定到期时间
	EmailVerified        bool       `gorm:"default:false" json:"email_verified"`        // 邮箱是否验证
	EmailVerifiedAt      *time.Time `gorm:"type:timestamp" json:"email_verified_at"`    // 邮箱验证时间
	VerificationToken    string     `gorm:"size:255;index" json:"-"`                    // 邮箱验证令牌
	VerificationExpires  *time.Time `gorm:"type:timestamp" json:"-"`                    // 验证令牌过期时间
	PasswordResetToken   string     `gorm:"size:255;index" json:"-"`                    // 密码重置令牌
	PasswordResetExpires *time.Time `gorm:"type:timestamp" json:"-"`                    // 密码重置过期时间
	TwoFactorEnabled     bool       `gorm:"default:false" json:"two_factor_enabled"`    // 是否启用双因素认证
	TwoFactorSecret      string     `gorm:"size:255" json:"-"`                          // 双因素认证密钥
	RememberToken        string     `gorm:"size:255;index" json:"-"`                    // 记住我令牌
	LastIPAddress        string     `gorm:"size:45" json:"last_ip_address"`             // 最后登录 IP
}

// UserShard0 分片0的用户表
type UserShard0 struct {
	UserBase
}

// TableName 指定表名
func (UserShard0) TableName() string {
	return "users_shard_0"
}

// BeforeCreate 创建前的钩子函数
func (u *UserShard0) BeforeCreate(tx *gorm.DB) error {
	if u.AccountType == "" {
		u.AccountType = AccountTypeLight
	}
	return nil
}

// UserShard1 分片1的用户表
type UserShard1 struct {
	UserBase
}

// TableName 指定表名
func (UserShard1) TableName() string {
	return "users_shard_1"
}

// BeforeCreate 创建前的钩子函数
func (u *UserShard1) BeforeCreate(tx *gorm.DB) error {
	if u.AccountType == "" {
		u.AccountType = AccountTypeLight
	}
	return nil
}

// User 通用用户接口（用于向后兼容和辅助函数）
type User struct {
	UserBase
}

// TableName 指定表名（默认表，用于向后兼容）
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前的钩子函数
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.AccountType == "" {
		u.AccountType = AccountTypeLight
	}
	return nil
}

// GetShardNumber 根据用户ID获取分片编号（用于分片策略）
func GetShardNumber(userID uint) int {
	return int(userID % 2)
}

// GetShardTableName 根据用户ID获取分片表名
func GetShardTableName(userID uint) string {
	shardNum := GetShardNumber(userID)
	return fmt.Sprintf("users_shard_%d", shardNum)
}
