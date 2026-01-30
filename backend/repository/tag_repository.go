package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// TagRepository 标签仓库接口
type TagRepository interface {
	// 基础 CRUD 操作
	Create(tag *entity.Tag) error
	GetByID(id uint) (*entity.Tag, error)
	GetByName(name string) (*entity.Tag, error)
	GetBySlug(slug string) (*entity.Tag, error)
	Update(tag *entity.Tag) error
	Delete(id uint) error

	// 查询操作
	List(limit, offset int) ([]entity.Tag, error)
	GetPopular(limit int) ([]entity.Tag, error)
	GetTrendingByCategory(limit int, category string) ([]TagWithCount, error)
	Search(keyword string, limit, offset int) ([]entity.Tag, error)

	// 统计操作
	Count() (int64, error)
	IncrementUsage(tagID uint) error
	DecrementUsage(tagID uint) error
}

// tagRepository 标签仓库实现
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建标签仓库实例
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// Create 创建标签
func (r *tagRepository) Create(tag *entity.Tag) error {
	return r.db.Create(tag).Error
}

// GetByID 根据ID获取标签
func (r *tagRepository) GetByID(id uint) (*entity.Tag, error) {
	var tag entity.Tag
	if err := r.db.First(&tag, id).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetByName 根据名称获取标签
func (r *tagRepository) GetByName(name string) (*entity.Tag, error) {
	var tag entity.Tag
	if err := r.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetBySlug 根据别名获取标签
func (r *tagRepository) GetBySlug(slug string) (*entity.Tag, error) {
	var tag entity.Tag
	if err := r.db.Where("slug = ?", slug).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// Update 更新标签
func (r *tagRepository) Update(tag *entity.Tag) error {
	return r.db.Save(tag).Error
}

// Delete 删除标签（软删除）
func (r *tagRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Tag{}, id).Error
}

// List 列出标签
func (r *tagRepository) List(limit, offset int) ([]entity.Tag, error) {
	var tags []entity.Tag
	query := r.db.Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// GetPopular 获取热门标签（按使用次数排序）
func (r *tagRepository) GetPopular(limit int) ([]entity.Tag, error) {
	var tags []entity.Tag
	query := r.db.Order("usage_count DESC").Limit(limit)
	if err := query.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// TagWithCount 标签及其关联的已发布内容数量（用于 trending）
type TagWithCount struct {
	ID        uint
	Name      string
	Slug      string
	PostCount int64
}

// GetTrendingByCategory 按内容分类获取热门标签（只统计 status=ready 的内容）。category: "light" 仅 light 内容，"all" 为全部
func (r *tagRepository) GetTrendingByCategory(limit int, category string) ([]TagWithCount, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	var results []TagWithCount
	query := r.db.Table("tags").
		Select("tags.id, tags.name, tags.slug, COUNT(DISTINCT contents.id) as post_count").
		Joins("INNER JOIN content_tags ON content_tags.tag_id = tags.id AND content_tags.deleted_at IS NULL").
		Joins("INNER JOIN contents ON contents.id = content_tags.content_id AND contents.deleted_at IS NULL AND contents.status = ?", entity.ContentStatusReady)
	if category == "light" {
		query = query.Where("contents.category = ?", entity.ContentCategoryLight)
	}
	query = query.Group("tags.id, tags.name, tags.slug").Order("post_count DESC").Limit(limit)
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// Search 搜索标签（按名称和描述）
func (r *tagRepository) Search(keyword string, limit, offset int) ([]entity.Tag, error) {
	var tags []entity.Tag
	query := r.db.Where("name ILIKE ? OR description ILIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).Offset(offset).Order("usage_count DESC")
	if err := query.Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// Count 统计标签数量
func (r *tagRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Tag{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// IncrementUsage 增加标签使用次数
func (r *tagRepository) IncrementUsage(tagID uint) error {
	return r.db.Model(&entity.Tag{}).
		Where("id = ?", tagID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error
}

// DecrementUsage 减少标签使用次数
func (r *tagRepository) DecrementUsage(tagID uint) error {
	return r.db.Model(&entity.Tag{}).
		Where("id = ? AND usage_count > 0", tagID).
		UpdateColumn("usage_count", gorm.Expr("usage_count - ?", 1)).Error
}
