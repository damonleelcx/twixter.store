package entity

import (
	"time"

	"gorm.io/gorm"
)

// Session 会话模型
type Session struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	UserID       uint   `gorm:"not null;index" json:"user_id"`              // 用户ID（用于分片查找）
	ShardNumber  int    `gorm:"not null;index" json:"shard_number"`         // 分片编号（0 或 1）
	AccessToken  string `gorm:"type:text;not null;index" json:"-"`          // 不序列化访问令牌
	RefreshToken string `gorm:"type:text;not null;index" json:"-"`          // 不序列化刷新令牌
	TokenType    string `gorm:"size:50;default:'Bearer'" json:"token_type"` // 令牌类型，默认为 Bearer

	// 过期时间
	AccessTokenExpiresAt  time.Time `gorm:"not null;index" json:"access_token_expires_at"`  // 访问令牌过期时间
	RefreshTokenExpiresAt time.Time `gorm:"not null;index" json:"refresh_token_expires_at"` // 刷新令牌过期时间

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除

	// 安全相关字段
	IPAddress    string     `gorm:"size:45" json:"ip_address"`           // 创建会话时的 IP 地址
	UserAgent    string     `gorm:"type:text" json:"user_agent"`         // 用户代理信息
	DeviceInfo   string     `gorm:"type:text" json:"device_info"`        // 设备信息
	LastUsedAt   time.Time  `gorm:"index" json:"last_used_at"`           // 最后使用时间
	IsActive     bool       `gorm:"default:true;index" json:"is_active"` // 会话是否活跃
	RevokedAt    *time.Time `gorm:"type:timestamp" json:"revoked_at"`    // 撤销时间
	RevokeReason string     `gorm:"size:255" json:"revoke_reason"`       // 撤销原因
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}

// BeforeCreate 创建前的钩子函数
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	// 自动计算分片编号（如果未设置）
	if s.UserID > 0 {
		s.ShardNumber = GetShardNumber(s.UserID)
	}
	// 设置最后使用时间为当前时间
	if s.LastUsedAt.IsZero() {
		s.LastUsedAt = time.Now()
	}
	return nil
}

// IsExpired 检查访问令牌是否过期
func (s *Session) IsExpired() bool {
	return time.Now().After(s.AccessTokenExpiresAt)
}

// IsRefreshTokenExpired 检查刷新令牌是否过期
func (s *Session) IsRefreshTokenExpired() bool {
	return time.Now().After(s.RefreshTokenExpiresAt)
}

// IsValid 检查会话是否有效（未过期且未撤销）
func (s *Session) IsValid() bool {
	return s.IsActive && !s.IsExpired() && s.RevokedAt == nil
}
