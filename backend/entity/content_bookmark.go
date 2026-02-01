package entity

import (
	"time"

	"gorm.io/gorm"
)

// ContentBookmark 用户内容书签（user_id + content_id 唯一）
type ContentBookmark struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"not null;uniqueIndex:idx_content_bookmarks_user_content" json:"user_id"`
	ContentID uint           `gorm:"not null;uniqueIndex:idx_content_bookmarks_user_content" json:"content_id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (ContentBookmark) TableName() string {
	return "content_bookmarks"
}
