package entity

import (
	"time"

	"gorm.io/gorm"
)

// ContentTag 内容标签关联模型（多对多关系）
type ContentTag struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	ContentID uint `gorm:"not null;index" json:"content_id"` // 内容ID
	TagID     uint `gorm:"not null;index" json:"tag_id"`     // 标签ID

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除

	// 关联关系（可选，用于预加载）
	Content Content `gorm:"foreignKey:ContentID" json:"content,omitempty"`
	Tag     Tag     `gorm:"foreignKey:TagID" json:"tag,omitempty"`
}

// TableName 指定表名
func (ContentTag) TableName() string {
	return "content_tags"
}

// BeforeCreate 创建前的钩子函数
func (ct *ContentTag) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加验证逻辑
	return nil
}
