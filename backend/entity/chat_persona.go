package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChatPersona 用户首次聊天时填写的名字与人格，存于 PostgreSQL
// 用于 Agent 人设：tone, speaking_style, boundaries, quirks, emotional_range
type ChatPersona struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	Name       string         `gorm:"size:255;not null" json:"name"`
	Personality datatypes.JSON `gorm:"type:jsonb;not null" json:"personality"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// PersonalitySchema 人格 JSON 结构（与前端约定）
type PersonalitySchema struct {
	Tone           string   `json:"tone"`
	SpeakingStyle  string   `json:"speaking_style"`
	Boundaries     string   `json:"boundaries"`
	Quirks         string `json:"quirks"`
	EmotionalRange string `json:"emotional_range"`
}
