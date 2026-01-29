package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// ContentBookmarkRepository 内容书签仓库接口
type ContentBookmarkRepository interface {
	Create(bookmark *entity.ContentBookmark) error
	Delete(userID, contentID uint) error
	Exists(userID, contentID uint) (bool, error)
	ListContentIDsByUserID(userID uint, limit, offset int) ([]uint, error)
}

type contentBookmarkRepository struct {
	db *gorm.DB
}

// NewContentBookmarkRepository 创建书签仓库实例
func NewContentBookmarkRepository(db *gorm.DB) ContentBookmarkRepository {
	return &contentBookmarkRepository{db: db}
}

func (r *contentBookmarkRepository) Create(bookmark *entity.ContentBookmark) error {
	return r.db.Create(bookmark).Error
}

func (r *contentBookmarkRepository) Delete(userID, contentID uint) error {
	return r.db.Where("user_id = ? AND content_id = ?", userID, contentID).Delete(&entity.ContentBookmark{}).Error
}

func (r *contentBookmarkRepository) Exists(userID, contentID uint) (bool, error) {
	var count int64
	err := r.db.Model(&entity.ContentBookmark{}).Where("user_id = ? AND content_id = ?", userID, contentID).Count(&count).Error
	return count > 0, err
}

func (r *contentBookmarkRepository) ListContentIDsByUserID(userID uint, limit, offset int) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&entity.ContentBookmark{}).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).Pluck("content_id", &ids).Error
	return ids, err
}
