package repository

import (
	"time"

	"backend/entity"

	"gorm.io/gorm"
)

// ChatPersonaRepository 聊天人格仓库（PostgreSQL）
type ChatPersonaRepository interface {
	GetByUserID(userID uint) (*entity.ChatPersona, error)
	Upsert(persona *entity.ChatPersona) error
}

type chatPersonaRepository struct {
	db *gorm.DB
}

// NewChatPersonaRepository 创建聊天人格仓库
func NewChatPersonaRepository(db *gorm.DB) ChatPersonaRepository {
	return &chatPersonaRepository{db: db}
}

// GetByUserID 按用户 ID 获取人格（唯一）
func (r *chatPersonaRepository) GetByUserID(userID uint) (*entity.ChatPersona, error) {
	var p entity.ChatPersona
	if err := r.db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Upsert 创建或更新人格（按 user_id 唯一）
func (r *chatPersonaRepository) Upsert(persona *entity.ChatPersona) error {
	existing, err := r.GetByUserID(persona.UserID)
	if err == nil {
		persona.ID = existing.ID
		persona.CreatedAt = existing.CreatedAt
		now := time.Now()
		return r.db.Model(existing).Updates(map[string]interface{}{
			"name": persona.Name, "personality": persona.Personality, "updated_at": now,
		}).Error
	}
	return r.db.Create(persona).Error
}
