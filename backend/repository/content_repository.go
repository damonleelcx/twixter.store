package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// ContentRepository 内容仓库接口
type ContentRepository interface {
	// 基础 CRUD 操作
	Create(content *entity.Content) error
	GetByID(id uint) (*entity.Content, error)
	Update(content *entity.Content) error
	Delete(id uint) error

	// 查询操作
	List(limit, offset int) ([]entity.Content, error)
	GetByUserID(userID uint, limit, offset int) ([]entity.Content, error)
	GetByType(contentType entity.ContentType, limit, offset int) ([]entity.Content, error)
	GetByStatus(status entity.ContentStatus, limit, offset int) ([]entity.Content, error)
	GetPublic(limit, offset int) ([]entity.Content, error)
	GetByCategory(category string, limit, offset int) ([]entity.Content, error)
	// ListFeedByCategory 按分类列出 feed 用内容（仅 status=ready）。sortBy: "view_count" 或 "created_at"（默认）
	ListFeedByCategory(category string, limit, offset int, sortBy string) ([]entity.Content, error)
	Search(keyword string, limit, offset int) ([]entity.Content, error)
	// SearchReady 模糊搜索 name/description，仅 status=ready；category 为空时不限分类。sortBy: "view_count" 或 "created_at"（默认）
	SearchReady(keyword string, category string, limit, offset int, sortBy string) ([]entity.Content, error)

	// 统计操作
	Count() (int64, error)
	CountByUserID(userID uint) (int64, error)
	CountByType(contentType entity.ContentType) (int64, error)
	CountByStatus(status entity.ContentStatus) (int64, error)
}

// contentRepository 内容仓库实现
type contentRepository struct {
	db *gorm.DB
}

// NewContentRepository 创建内容仓库实例
func NewContentRepository(db *gorm.DB) ContentRepository {
	return &contentRepository{db: db}
}

// Create 创建内容
func (r *contentRepository) Create(content *entity.Content) error {
	return r.db.Create(content).Error
}

// GetByID 根据ID获取内容
func (r *contentRepository) GetByID(id uint) (*entity.Content, error) {
	var content entity.Content
	if err := r.db.First(&content, id).Error; err != nil {
		return nil, err
	}
	return &content, nil
}

// Update 更新内容
func (r *contentRepository) Update(content *entity.Content) error {
	return r.db.Save(content).Error
}

// Delete 删除内容（软删除）
func (r *contentRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Content{}, id).Error
}

// List 列出内容
func (r *contentRepository) List(limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// GetByUserID 根据用户ID获取内容列表
func (r *contentRepository) GetByUserID(userID uint, limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("uploaded_by = ?", userID).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// GetByType 根据类型获取内容列表
func (r *contentRepository) GetByType(contentType entity.ContentType, limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("type = ?", contentType).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// GetByStatus 根据状态获取内容列表
func (r *contentRepository) GetByStatus(status entity.ContentStatus, limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("status = ?", status).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// GetPublic 获取公开内容列表
func (r *contentRepository) GetPublic(limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("is_public = ? AND status = ?", true, entity.ContentStatusReady).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// GetByCategory 根据分类获取内容列表
func (r *contentRepository) GetByCategory(category string, limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("category = ?", category).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// orderByClause 根据 sortBy 返回 ORDER 子句；仅支持 view_count、created_at，默认 created_at DESC
func orderByClause(sortBy string) string {
	switch sortBy {
	case "view_count":
		return "view_count DESC"
	default:
		return "created_at DESC"
	}
}

// ListFeedByCategory 按分类列出 feed 用内容（仅 status=ready）
func (r *contentRepository) ListFeedByCategory(category string, limit, offset int, sortBy string) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("category = ? AND status = ?", category, entity.ContentStatusReady).
		Limit(limit).Offset(offset).Order(orderByClause(sortBy))
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// Search 搜索内容（按名称和描述）
func (r *contentRepository) Search(keyword string, limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("name ILIKE ? OR description ILIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// SearchReady 模糊搜索 name/description，仅 status=ready；category 为空时不限分类
func (r *contentRepository) SearchReady(keyword string, category string, limit, offset int, sortBy string) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Where("status = ?", entity.ContentStatusReady)
	if keyword != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category != "" && (category == "light" || category == "dark") {
		query = query.Where("category = ?", category)
	}
	query = query.Limit(limit).Offset(offset).Order(orderByClause(sortBy))
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// Count 统计内容数量
func (r *contentRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Content{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByUserID 统计用户的内容数量
func (r *contentRepository) CountByUserID(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Content{}).Where("uploaded_by = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByType 统计指定类型的内容数量
func (r *contentRepository) CountByType(contentType entity.ContentType) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Content{}).Where("type = ?", contentType).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountByStatus 统计指定状态的内容数量
func (r *contentRepository) CountByStatus(status entity.ContentStatus) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Content{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
