package entity

import (
	"time"

	"gorm.io/gorm"
)

// PermissionAction 权限操作类型枚举
type PermissionAction string

const (
	ActionView   PermissionAction = "view"   // 查看
	ActionCreate PermissionAction = "create" // 创建
	ActionEdit   PermissionAction = "edit"   // 编辑
	ActionDelete PermissionAction = "delete" // 删除
)

// Permission 权限模型
type Permission struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	Name        string          `gorm:"uniqueIndex;not null;size:100" json:"name"`        // 权限名称，如 "view_users"
	Description string          `gorm:"type:text" json:"description"`                    // 权限描述
	Resource    string          `gorm:"not null;size:100;index" json:"resource"`         // 资源类型，如 "users", "products", "orders"
	Action      PermissionAction `gorm:"type:varchar(20);not null;index" json:"action"`  // 操作类型：view, create, edit, delete
	
	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
	
	// 元数据
	IsSystem bool `gorm:"default:false" json:"is_system"` // 是否为系统权限（系统权限不可删除）
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}

// BeforeCreate 创建前的钩子函数
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加验证逻辑
	return nil
}

// GetFullName 获取完整权限名称（resource:action 格式）
func (p *Permission) GetFullName() string {
	return p.Resource + ":" + string(p.Action)
}
