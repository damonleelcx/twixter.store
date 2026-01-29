package entity

import (
	"time"

	"gorm.io/gorm"
)

// FileProcessingStage 文件处理阶段枚举
type FileProcessingStage string

const (
	StageUploaded           FileProcessingStage = "uploaded"            // 已上传到S3
	StageGifGenerated       FileProcessingStage = "gif_generated"       // GIF已生成
	StageTranscoded         FileProcessingStage = "transcoded"          // 已转码
	StageTranscodedUploaded FileProcessingStage = "transcoded_uploaded" // 转码文件已上传
	StageOriginalDeleted    FileProcessingStage = "original_deleted"    // 原始文件已删除
	StageCompleted          FileProcessingStage = "completed"           // 处理完成
	StageFailed             FileProcessingStage = "failed"              // 处理失败
)

// ContentFile 内容文件模型（用于跟踪每个文件的处理状态）
type ContentFile struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ContentID uint   `gorm:"not null;index" json:"content_id"`     // 关联的内容ID
	FileName  string `gorm:"not null;size:255" json:"file_name"`   // 原始文件名
	FileIndex int    `gorm:"not null;default:0" json:"file_index"` // 文件索引（0-9）

	// 原始文件信息
	OriginalFilePath string `gorm:"type:text;not null" json:"original_file_path"` // S3原始文件路径
	OriginalFileURL  string `gorm:"type:text" json:"original_file_url"`           // 原始文件URL
	OriginalFileSize int64  `gorm:"default:0" json:"original_file_size"`          // 原始文件大小
	MimeType         string `gorm:"size:100" json:"mime_type"`                    // MIME类型

	// GIF文件信息
	GifFilePath string `gorm:"type:text" json:"gif_file_path"` // GIF文件S3路径
	GifFileURL  string `gorm:"type:text" json:"gif_file_url"`  // GIF文件URL
	GifFileSize int64  `gorm:"default:0" json:"gif_file_size"` // GIF文件大小

	// 转码文件信息
	TranscodedFilePath string `gorm:"type:text" json:"transcoded_file_path"` // 转码文件S3路径
	TranscodedFileURL  string `gorm:"type:text" json:"transcoded_file_url"`  // 转码文件URL
	TranscodedFileSize int64  `gorm:"default:0" json:"transcoded_file_size"` // 转码文件大小

	// 处理状态
	Stage           FileProcessingStage `gorm:"type:varchar(30);not null;default:'uploaded';index" json:"stage"` // 当前处理阶段
	ProcessingError string              `gorm:"type:text" json:"processing_error"`                               // 处理错误信息

	// 媒体属性
	Width    *int     `gorm:"default:0" json:"width"`    // 宽度（像素）
	Height   *int     `gorm:"default:0" json:"height"`   // 高度（像素）
	Duration *float64 `gorm:"default:0" json:"duration"` // 时长（秒）

	// 时间戳
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ProcessedAt *time.Time     `gorm:"type:timestamp" json:"processed_at"` // 处理完成时间
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`                     // 软删除
}

// TableName 指定表名
func (ContentFile) TableName() string {
	return "content_files"
}

// BeforeCreate 创建前的钩子函数
func (cf *ContentFile) BeforeCreate(tx *gorm.DB) error {
	// 设置默认阶段
	if cf.Stage == "" {
		cf.Stage = StageUploaded
	}
	return nil
}

// IsCompleted 检查文件处理是否完成
func (cf *ContentFile) IsCompleted() bool {
	return cf.Stage == StageCompleted
}

// IsFailed 检查文件处理是否失败
func (cf *ContentFile) IsFailed() bool {
	return cf.Stage == StageFailed
}

// CanGenerateGif 检查是否可以生成GIF
func (cf *ContentFile) CanGenerateGif() bool {
	return cf.Stage == StageUploaded
}

// CanTranscode 检查是否可以转码
func (cf *ContentFile) CanTranscode() bool {
	return cf.Stage == StageGifGenerated
}

// CanUploadTranscoded 检查是否可以上传转码文件
func (cf *ContentFile) CanUploadTranscoded() bool {
	return cf.Stage == StageTranscoded
}

// CanDeleteOriginal 检查是否可以删除原始文件
func (cf *ContentFile) CanDeleteOriginal() bool {
	return cf.Stage == StageTranscodedUploaded
}
