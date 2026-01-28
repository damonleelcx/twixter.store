package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// ContentFileRepository 内容文件仓库接口
type ContentFileRepository interface {
	// 基础 CRUD 操作
	Create(file *entity.ContentFile) error
	GetByID(id uint) (*entity.ContentFile, error)
	Update(file *entity.ContentFile) error
	Delete(id uint) error
	
	// 查询操作
	GetByContentID(contentID uint) ([]*entity.ContentFile, error)
	GetByStage(stage entity.FileProcessingStage, limit, offset int) ([]*entity.ContentFile, error)
}

// contentFileRepository 内容文件仓库实现
type contentFileRepository struct {
	db *gorm.DB
}

// NewContentFileRepository 创建内容文件仓库实例
func NewContentFileRepository(db *gorm.DB) ContentFileRepository {
	return &contentFileRepository{db: db}
}

// Create 创建文件记录
func (r *contentFileRepository) Create(file *entity.ContentFile) error {
	return r.db.Create(file).Error
}

// GetByID 根据ID获取文件记录
func (r *contentFileRepository) GetByID(id uint) (*entity.ContentFile, error) {
	var file entity.ContentFile
	if err := r.db.First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// Update 更新文件记录
func (r *contentFileRepository) Update(file *entity.ContentFile) error {
	return r.db.Save(file).Error
}

// Delete 删除文件记录（软删除）
func (r *contentFileRepository) Delete(id uint) error {
	return r.db.Delete(&entity.ContentFile{}, id).Error
}

// GetByContentID 根据内容ID获取所有文件
func (r *contentFileRepository) GetByContentID(contentID uint) ([]*entity.ContentFile, error) {
	var files []*entity.ContentFile
	if err := r.db.Where("content_id = ?", contentID).Order("file_index ASC").Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// GetByStage 根据处理阶段获取文件列表
func (r *contentFileRepository) GetByStage(stage entity.FileProcessingStage, limit, offset int) ([]*entity.ContentFile, error) {
	var files []*entity.ContentFile
	query := r.db.Where("stage = ?", stage).
		Limit(limit).Offset(offset).Order("created_at ASC")
	if err := query.Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}
