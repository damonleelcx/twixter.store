package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// ContentTagRepository 内容标签关联仓库接口
type ContentTagRepository interface {
	// 基础 CRUD 操作
	Create(contentTag *entity.ContentTag) error
	GetByID(id uint) (*entity.ContentTag, error)
	Delete(id uint) error

	// 关联操作
	AddTagToContent(contentID, tagID uint) error
	RemoveTagFromContent(contentID, tagID uint) error
	GetTagsByContentID(contentID uint) ([]entity.Tag, error)
	GetContentsByTagID(tagID uint, limit, offset int) ([]entity.Content, error)
	// GetReadyContentsByTagID 获取该标签下 status=ready 的内容（用于 feed）。createdAtDesc: 按上传时间排序时 true=新在前 false=旧在前
	GetReadyContentsByTagID(tagID uint, limit, offset int, sortBy string, freeOnly bool, createdAtDesc bool) ([]entity.Content, error)

	// 批量操作
	AddTagsToContent(contentID uint, tagIDs []uint) error
	RemoveAllTagsFromContent(contentID uint) error
	ReplaceContentTags(contentID uint, tagIDs []uint) error

	// 查询操作
	GetByContentAndTag(contentID, tagID uint) (*entity.ContentTag, error)
	CountTagsByContentID(contentID uint) (int64, error)
	CountContentsByTagID(tagID uint) (int64, error)
}

// contentTagRepository 内容标签关联仓库实现
type contentTagRepository struct {
	db *gorm.DB
}

// NewContentTagRepository 创建内容标签关联仓库实例
func NewContentTagRepository(db *gorm.DB) ContentTagRepository {
	return &contentTagRepository{db: db}
}

// Create 创建内容标签关联
func (r *contentTagRepository) Create(contentTag *entity.ContentTag) error {
	return r.db.Create(contentTag).Error
}

// GetByID 根据ID获取内容标签关联
func (r *contentTagRepository) GetByID(id uint) (*entity.ContentTag, error) {
	var contentTag entity.ContentTag
	if err := r.db.First(&contentTag, id).Error; err != nil {
		return nil, err
	}
	return &contentTag, nil
}

// Delete 删除内容标签关联（软删除）
func (r *contentTagRepository) Delete(id uint) error {
	return r.db.Delete(&entity.ContentTag{}, id).Error
}

// AddTagToContent 为内容添加标签
func (r *contentTagRepository) AddTagToContent(contentID, tagID uint) error {
	// 检查是否已存在
	var existing entity.ContentTag
	err := r.db.Where("content_id = ? AND tag_id = ?", contentID, tagID).First(&existing).Error
	if err == nil {
		// 已存在，不重复添加
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 创建新的关联
	contentTag := &entity.ContentTag{
		ContentID: contentID,
		TagID:     tagID,
	}
	if err := r.db.Create(contentTag).Error; err != nil {
		return err
	}

	// 增加标签使用次数
	tagRepo := NewTagRepository(r.db)
	return tagRepo.IncrementUsage(tagID)
}

// RemoveTagFromContent 从内容中移除标签
func (r *contentTagRepository) RemoveTagFromContent(contentID, tagID uint) error {
	// 删除关联
	if err := r.db.Where("content_id = ? AND tag_id = ?", contentID, tagID).
		Delete(&entity.ContentTag{}).Error; err != nil {
		return err
	}

	// 减少标签使用次数
	tagRepo := NewTagRepository(r.db)
	return tagRepo.DecrementUsage(tagID)
}

// GetTagsByContentID 获取内容的所有标签
func (r *contentTagRepository) GetTagsByContentID(contentID uint) ([]entity.Tag, error) {
	var tags []entity.Tag
	if err := r.db.Table("tags").
		Joins("INNER JOIN content_tags ON tags.id = content_tags.tag_id").
		Where("content_tags.content_id = ? AND content_tags.deleted_at IS NULL", contentID).
		Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// GetContentsByTagID 获取标签的所有内容
func (r *contentTagRepository) GetContentsByTagID(tagID uint, limit, offset int) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Table("contents").
		Joins("INNER JOIN content_tags ON contents.id = content_tags.content_id").
		Where("content_tags.tag_id = ? AND content_tags.deleted_at IS NULL", tagID).
		Limit(limit).Offset(offset).Order("contents.created_at DESC")
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// orderByContentsClause 根据 sortBy 返回 contents 表的 ORDER 子句
func orderByContentsClause(sortBy string, createdAtDesc bool) string {
	switch sortBy {
	case "view_count":
		return "contents.view_count DESC"
	default:
		if createdAtDesc {
			return "contents.created_at DESC"
		}
		return "contents.created_at ASC"
	}
}

// GetReadyContentsByTagID 获取该标签下 status=ready 的内容（用于 feed）；freeOnly 为 true 时仅返回 price=0 的免费内容
func (r *contentTagRepository) GetReadyContentsByTagID(tagID uint, limit, offset int, sortBy string, freeOnly bool, createdAtDesc bool) ([]entity.Content, error) {
	var contents []entity.Content
	query := r.db.Table("contents").
		Joins("INNER JOIN content_tags ON contents.id = content_tags.content_id").
		Where("content_tags.tag_id = ? AND content_tags.deleted_at IS NULL AND contents.status = ? AND contents.deleted_at IS NULL", tagID, entity.ContentStatusReady)
	if freeOnly {
		query = query.Where("contents.price = ?", 0)
	}
	query = query.Limit(limit).Offset(offset).Order(orderByContentsClause(sortBy, createdAtDesc))
	if err := query.Find(&contents).Error; err != nil {
		return nil, err
	}
	return contents, nil
}

// AddTagsToContent 为内容批量添加标签
func (r *contentTagRepository) AddTagsToContent(contentID uint, tagIDs []uint) error {
	for _, tagID := range tagIDs {
		if err := r.AddTagToContent(contentID, tagID); err != nil {
			return err
		}
	}
	return nil
}

// RemoveAllTagsFromContent 移除内容的所有标签
func (r *contentTagRepository) RemoveAllTagsFromContent(contentID uint) error {
	// 获取所有关联的标签ID
	var contentTags []entity.ContentTag
	if err := r.db.Where("content_id = ?", contentID).Find(&contentTags).Error; err != nil {
		return err
	}

	// 删除所有关联
	if err := r.db.Where("content_id = ?", contentID).Delete(&entity.ContentTag{}).Error; err != nil {
		return err
	}

	// 减少所有标签的使用次数
	tagRepo := NewTagRepository(r.db)
	for _, ct := range contentTags {
		tagRepo.DecrementUsage(ct.TagID)
	}

	return nil
}

// ReplaceContentTags 替换内容的所有标签
func (r *contentTagRepository) ReplaceContentTags(contentID uint, tagIDs []uint) error {
	// 先移除所有现有标签
	if err := r.RemoveAllTagsFromContent(contentID); err != nil {
		return err
	}

	// 添加新标签
	return r.AddTagsToContent(contentID, tagIDs)
}

// GetByContentAndTag 根据内容和标签获取关联
func (r *contentTagRepository) GetByContentAndTag(contentID, tagID uint) (*entity.ContentTag, error) {
	var contentTag entity.ContentTag
	if err := r.db.Where("content_id = ? AND tag_id = ?", contentID, tagID).
		First(&contentTag).Error; err != nil {
		return nil, err
	}
	return &contentTag, nil
}

// CountTagsByContentID 统计内容的标签数量
func (r *contentTagRepository) CountTagsByContentID(contentID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.ContentTag{}).
		Where("content_id = ?", contentID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountContentsByTagID 统计标签关联的内容数量
func (r *contentTagRepository) CountContentsByTagID(tagID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.ContentTag{}).
		Where("tag_id = ?", tagID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
