package entity

import (
	"time"

	"gorm.io/gorm"
)

// ContentType 内容类型枚举
type ContentType string

const (
	ContentTypeVideo ContentType = "video" // 视频
	ContentTypeImage ContentType = "image" // 图片
)

// ContentStatus 内容状态枚举
type ContentStatus string

const (
	ContentStatusPending    ContentStatus = "pending"    // 待处理
	ContentStatusProcessing ContentStatus = "processing" // 处理中
	ContentStatusReady      ContentStatus = "ready"      // 就绪
	ContentStatusFailed     ContentStatus = "failed"     // 失败
)

// Content 内容模型
type Content struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	Name        string        `gorm:"not null;size:255;index" json:"name"`                             // 内容名称
	Description string        `gorm:"type:text" json:"description"`                                    // 内容描述
	Type        ContentType   `gorm:"type:varchar(20);not null;index" json:"type"`                     // 内容类型：video, image
	Status      ContentStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"` // 内容状态
	UploadedBy  uint          `gorm:"not null;index" json:"uploaded_by"`                               // 上传者用户ID
	ShardNumber int           `gorm:"not null;index" json:"shard_number"`                              // 分片编号（0 或 1）

	// 文件相关字段
	FilePath     string `gorm:"type:text;not null" json:"file_path"` // 文件存储路径
	FileURL      string `gorm:"type:text" json:"file_url"`           // 文件访问URL
	FileSize     int64  `gorm:"default:0" json:"file_size"`          // 文件大小（字节）
	MimeType     string `gorm:"size:100" json:"mime_type"`           // MIME类型
	ThumbnailURL string `gorm:"type:text" json:"thumbnail_url"`      // 缩略图URL（主要用于视频）

	// 媒体属性（图片和视频）
	Width    *int     `gorm:"default:0" json:"width"`    // 宽度（像素）
	Height   *int     `gorm:"default:0" json:"height"`   // 高度（像素）
	Duration *float64 `gorm:"default:0" json:"duration"` // 时长（秒，主要用于视频）

	// 元数据
	Category  string `gorm:"size:100;index" json:"category"`       // 分类
	IsPublic  bool   `gorm:"default:false;index" json:"is_public"` // 是否公开
	ViewCount int64  `gorm:"default:0" json:"view_count"`          // 查看次数
	LikeCount int64  `gorm:"default:0" json:"like_count"`          // 点赞次数

	// 处理相关
	ProcessingError string     `gorm:"type:text" json:"processing_error"`  // 处理错误信息
	ProcessedAt     *time.Time `gorm:"type:timestamp" json:"processed_at"` // 处理完成时间

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// TableName 指定表名
func (Content) TableName() string {
	return "contents"
}

// BeforeCreate 创建前的钩子函数
func (c *Content) BeforeCreate(tx *gorm.DB) error {
	// 自动计算分片编号（如果未设置）
	if c.UploadedBy > 0 {
		c.ShardNumber = GetShardNumber(c.UploadedBy)
	}
	// 设置默认状态
	if c.Status == "" {
		c.Status = ContentStatusPending
	}
	return nil
}

// IsVideo 检查是否为视频类型
func (c *Content) IsVideo() bool {
	return c.Type == ContentTypeVideo
}

// IsImage 检查是否为图片类型
func (c *Content) IsImage() bool {
	return c.Type == ContentTypeImage
}

// IsReady 检查内容是否已就绪
func (c *Content) IsReady() bool {
	return c.Status == ContentStatusReady
}

// IsProcessing 检查内容是否正在处理
func (c *Content) IsProcessing() bool {
	return c.Status == ContentStatusProcessing
}

// IsFailed 检查内容处理是否失败
func (c *Content) IsFailed() bool {
	return c.Status == ContentStatusFailed
}
