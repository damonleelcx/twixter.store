package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// UserPermissionRepository 用户权限关联仓库接口
type UserPermissionRepository interface {
	// 基础 CRUD 操作
	Create(userPermission *entity.UserPermission) error
	GetByID(id uint) (*entity.UserPermission, error)
	Delete(id uint) error

	// 关联操作
	GrantPermission(userID, permissionID uint) error
	RevokePermission(userID, permissionID uint) error
	GetUserPermissions(userID uint) ([]entity.Permission, error)
	GetUsersByPermission(permissionID uint, limit, offset int) ([]uint, error)

	// 批量操作
	GrantPermissions(userID uint, permissionIDs []uint) error
	RevokeAllPermissions(userID uint) error
	ReplaceUserPermissions(userID uint, permissionIDs []uint) error

	// 权限检查
	HasPermission(userID uint, permissionName string) (bool, error)
	HasPermissionByResourceAction(userID uint, resource string, action entity.PermissionAction) (bool, error)
	HasAnyPermission(userID uint, permissionNames []string) (bool, error)
	HasAllPermissions(userID uint, permissionNames []string) (bool, error)

	// 查询操作
	GetByUserAndPermission(userID, permissionID uint) (*entity.UserPermission, error)
	CountUserPermissions(userID uint) (int64, error)
	CountUsersByPermission(permissionID uint) (int64, error)
}

// userPermissionRepository 用户权限关联仓库实现
type userPermissionRepository struct {
	db *gorm.DB
}

// NewUserPermissionRepository 创建用户权限关联仓库实例
func NewUserPermissionRepository(db *gorm.DB) UserPermissionRepository {
	return &userPermissionRepository{db: db}
}

// Create 创建用户权限关联
func (r *userPermissionRepository) Create(userPermission *entity.UserPermission) error {
	return r.db.Create(userPermission).Error
}

// GetByID 根据ID获取用户权限关联
func (r *userPermissionRepository) GetByID(id uint) (*entity.UserPermission, error) {
	var userPermission entity.UserPermission
	if err := r.db.First(&userPermission, id).Error; err != nil {
		return nil, err
	}
	return &userPermission, nil
}

// Delete 删除用户权限关联（软删除）
func (r *userPermissionRepository) Delete(id uint) error {
	return r.db.Delete(&entity.UserPermission{}, id).Error
}

// GrantPermission 授予用户权限
func (r *userPermissionRepository) GrantPermission(userID, permissionID uint) error {
	// 检查是否已存在
	var existing entity.UserPermission
	err := r.db.Where("user_id = ? AND permission_id = ?", userID, permissionID).
		First(&existing).Error
	if err == nil {
		// 已存在，不重复添加
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 创建新的关联
	userPermission := &entity.UserPermission{
		UserID:       userID,
		PermissionID: permissionID,
	}
	return r.db.Create(userPermission).Error
}

// RevokePermission 撤销用户权限
func (r *userPermissionRepository) RevokePermission(userID, permissionID uint) error {
	return r.db.Where("user_id = ? AND permission_id = ?", userID, permissionID).
		Delete(&entity.UserPermission{}).Error
}

// GetUserPermissions 获取用户的所有权限
func (r *userPermissionRepository) GetUserPermissions(userID uint) ([]entity.Permission, error) {
	var permissions []entity.Permission
	if err := r.db.Table("permissions").
		Joins("INNER JOIN user_permissions ON permissions.id = user_permissions.permission_id").
		Where("user_permissions.user_id = ? AND user_permissions.deleted_at IS NULL", userID).
		Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetUsersByPermission 获取拥有指定权限的所有用户ID
func (r *userPermissionRepository) GetUsersByPermission(permissionID uint, limit, offset int) ([]uint, error) {
	var userIDs []uint
	query := r.db.Model(&entity.UserPermission{}).
		Where("permission_id = ?", permissionID).
		Limit(limit).Offset(offset).
		Pluck("user_id", &userIDs)
	if err := query.Error; err != nil {
		return nil, err
	}
	return userIDs, nil
}

// GrantPermissions 批量授予用户权限
func (r *userPermissionRepository) GrantPermissions(userID uint, permissionIDs []uint) error {
	for _, permissionID := range permissionIDs {
		if err := r.GrantPermission(userID, permissionID); err != nil {
			return err
		}
	}
	return nil
}

// RevokeAllPermissions 撤销用户的所有权限
func (r *userPermissionRepository) RevokeAllPermissions(userID uint) error {
	return r.db.Where("user_id = ?", userID).
		Delete(&entity.UserPermission{}).Error
}

// ReplaceUserPermissions 替换用户的所有权限
func (r *userPermissionRepository) ReplaceUserPermissions(userID uint, permissionIDs []uint) error {
	// 先撤销所有现有权限
	if err := r.RevokeAllPermissions(userID); err != nil {
		return err
	}

	// 授予新权限
	return r.GrantPermissions(userID, permissionIDs)
}

// HasPermission 检查用户是否拥有指定权限
func (r *userPermissionRepository) HasPermission(userID uint, permissionName string) (bool, error) {
	var count int64
	if err := r.db.Table("user_permissions").
		Joins("INNER JOIN permissions ON user_permissions.permission_id = permissions.id").
		Where("user_permissions.user_id = ? AND permissions.name = ? AND user_permissions.deleted_at IS NULL",
			userID, permissionName).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasPermissionByResourceAction 检查用户是否拥有指定资源和操作的权限
func (r *userPermissionRepository) HasPermissionByResourceAction(userID uint, resource string, action entity.PermissionAction) (bool, error) {
	var count int64
	if err := r.db.Table("user_permissions").
		Joins("INNER JOIN permissions ON user_permissions.permission_id = permissions.id").
		Where("user_permissions.user_id = ? AND permissions.resource = ? AND permissions.action = ? AND user_permissions.deleted_at IS NULL",
			userID, resource, action).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasAnyPermission 检查用户是否拥有任意一个权限
func (r *userPermissionRepository) HasAnyPermission(userID uint, permissionNames []string) (bool, error) {
	if len(permissionNames) == 0 {
		return false, nil
	}

	var count int64
	if err := r.db.Table("user_permissions").
		Joins("INNER JOIN permissions ON user_permissions.permission_id = permissions.id").
		Where("user_permissions.user_id = ? AND permissions.name IN ? AND user_permissions.deleted_at IS NULL",
			userID, permissionNames).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasAllPermissions 检查用户是否拥有所有权限
func (r *userPermissionRepository) HasAllPermissions(userID uint, permissionNames []string) (bool, error) {
	if len(permissionNames) == 0 {
		return true, nil
	}

	var count int64
	if err := r.db.Table("user_permissions").
		Joins("INNER JOIN permissions ON user_permissions.permission_id = permissions.id").
		Where("user_permissions.user_id = ? AND permissions.name IN ? AND user_permissions.deleted_at IS NULL",
			userID, permissionNames).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == int64(len(permissionNames)), nil
}

// GetByUserAndPermission 根据用户和权限获取关联
func (r *userPermissionRepository) GetByUserAndPermission(userID, permissionID uint) (*entity.UserPermission, error) {
	var userPermission entity.UserPermission
	if err := r.db.Where("user_id = ? AND permission_id = ?", userID, permissionID).
		First(&userPermission).Error; err != nil {
		return nil, err
	}
	return &userPermission, nil
}

// CountUserPermissions 统计用户的权限数量
func (r *userPermissionRepository) CountUserPermissions(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.UserPermission{}).
		Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountUsersByPermission 统计拥有指定权限的用户数量
func (r *userPermissionRepository) CountUsersByPermission(permissionID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.UserPermission{}).
		Where("permission_id = ?", permissionID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
