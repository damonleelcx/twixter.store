package entity

import (
	"time"

	"gorm.io/gorm"
)

// Tag 标签模型
type Tag struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"uniqueIndex;not null;size:100" json:"name"` // 标签名称
	Slug        string `gorm:"uniqueIndex;not null;size:100" json:"slug"` // 标签别名（URL友好）
	Description string `gorm:"type:text" json:"description"`              // 标签描述
	Color       string `gorm:"size:7" json:"color"`                       // 标签颜色（十六进制，如 #FF5733）
	UsageCount  int64  `gorm:"default:0;index" json:"usage_count"`        // 使用次数

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}

// BeforeCreate 创建前的钩子函数
func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加验证逻辑
	return nil
}
