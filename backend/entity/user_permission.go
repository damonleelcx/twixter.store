package entity

import (
	"time"

	"gorm.io/gorm"
)

// UserPermission 用户权限关联模型（多对多关系）
type UserPermission struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	UserID       uint `gorm:"not null;index" json:"user_id"`       // 用户ID（用于分片查找）
	ShardNumber  int  `gorm:"not null;index" json:"shard_number"`  // 分片编号（0 或 1）
	PermissionID uint `gorm:"not null;index" json:"permission_id"` // 权限ID

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除

	// 关联关系（可选，用于预加载）
	Permission Permission `gorm:"foreignKey:PermissionID" json:"permission,omitempty"`
}

// TableName 指定表名
func (UserPermission) TableName() string {
	return "user_permissions"
}

// BeforeCreate 创建前的钩子函数
func (up *UserPermission) BeforeCreate(tx *gorm.DB) error {
	// 自动计算分片编号（如果未设置）
	if up.UserID > 0 {
		up.ShardNumber = GetShardNumber(up.UserID)
	}
	return nil
}

// HasPermission 检查用户是否拥有指定权限（辅助函数，用于业务逻辑）
func HasPermission(userID uint, permissionName string) bool {
	// 这个函数需要在 repository 或 service 层实现
	// 这里只是占位符
	return false
}

// HasPermissionByResourceAction 检查用户是否拥有指定资源和操作的权限
func HasPermissionByResourceAction(userID uint, resource string, action PermissionAction) bool {
	// 这个函数需要在 repository 或 service 层实现
	// 这里只是占位符
	return false
}
