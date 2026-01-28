package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// PermissionRepository 权限仓库接口
type PermissionRepository interface {
	// 基础 CRUD 操作
	Create(permission *entity.Permission) error
	GetByID(id uint) (*entity.Permission, error)
	GetByName(name string) (*entity.Permission, error)
	GetByResourceAndAction(resource string, action entity.PermissionAction) (*entity.Permission, error)
	Update(permission *entity.Permission) error
	Delete(id uint) error

	// 查询操作
	List(limit, offset int) ([]entity.Permission, error)
	GetByResource(resource string) ([]entity.Permission, error)
	GetByAction(action entity.PermissionAction) ([]entity.Permission, error)
	GetSystemPermissions() ([]entity.Permission, error)
	Search(keyword string, limit, offset int) ([]entity.Permission, error)

	// 统计操作
	Count() (int64, error)
	CountByResource(resource string) (int64, error)
}

// permissionRepository 权限仓库实现
type permissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 创建权限仓库实例
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

// Create 创建权限
func (r *permissionRepository) Create(permission *entity.Permission) error {
	return r.db.Create(permission).Error
}

// GetByID 根据ID获取权限
func (r *permissionRepository) GetByID(id uint) (*entity.Permission, error) {
	var permission entity.Permission
	if err := r.db.First(&permission, id).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetByName 根据名称获取权限
func (r *permissionRepository) GetByName(name string) (*entity.Permission, error) {
	var permission entity.Permission
	if err := r.db.Where("name = ?", name).First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetByResourceAndAction 根据资源和操作获取权限
func (r *permissionRepository) GetByResourceAndAction(resource string, action entity.PermissionAction) (*entity.Permission, error) {
	var permission entity.Permission
	if err := r.db.Where("resource = ? AND action = ?", resource, action).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// Update 更新权限
func (r *permissionRepository) Update(permission *entity.Permission) error {
	return r.db.Save(permission).Error
}

// Delete 删除权限（软删除，系统权限不能删除）
func (r *permissionRepository) Delete(id uint) error {
	var permission entity.Permission
	if err := r.db.First(&permission, id).Error; err != nil {
		return err
	}
	if permission.IsSystem {
		return gorm.ErrRecordNotFound // 系统权限不能删除
	}
	return r.db.Delete(&permission).Error
}

// List 列出权限
func (r *permissionRepository) List(limit, offset int) ([]entity.Permission, error) {
	var permissions []entity.Permission
	query := r.db.Limit(limit).Offset(offset).Order("resource, action")
	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetByResource 根据资源获取权限列表
func (r *permissionRepository) GetByResource(resource string) ([]entity.Permission, error) {
	var permissions []entity.Permission
	if err := r.db.Where("resource = ?", resource).
		Order("action").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetByAction 根据操作获取权限列表
func (r *permissionRepository) GetByAction(action entity.PermissionAction) ([]entity.Permission, error) {
	var permissions []entity.Permission
	if err := r.db.Where("action = ?", action).
		Order("resource").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetSystemPermissions 获取系统权限列表
func (r *permissionRepository) GetSystemPermissions() ([]entity.Permission, error) {
	var permissions []entity.Permission
	if err := r.db.Where("is_system = ?", true).
		Order("resource, action").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// Search 搜索权限（按名称、描述和资源）
func (r *permissionRepository) Search(keyword string, limit, offset int) ([]entity.Permission, error) {
	var permissions []entity.Permission
	query := r.db.Where("name ILIKE ? OR description ILIKE ? OR resource ILIKE ?",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).Offset(offset).Order("resource, action")
	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// Count 统计权限数量
func (r *permissionRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Permission{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByResource 统计指定资源的权限数量
func (r *permissionRepository) CountByResource(resource string) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Permission{}).
		Where("resource = ?", resource).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
