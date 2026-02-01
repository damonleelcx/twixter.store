package repository

import (
	"backend/entity"
	"time"

	"gorm.io/gorm"
)

// SessionRepository 会话仓库接口
type SessionRepository interface {
	// 基础 CRUD 操作
	Create(session *entity.Session) error
	GetByID(id uint) (*entity.Session, error)
	GetByAccessToken(accessToken string) (*entity.Session, error)
	GetByRefreshToken(refreshToken string) (*entity.Session, error)
	Update(session *entity.Session) error
	Delete(id uint) error

	// 查询操作
	GetByUserID(userID uint, limit, offset int) ([]entity.Session, error)
	GetActiveByUserID(userID uint) ([]entity.Session, error)
	GetExpiredSessions() ([]entity.Session, error)

	// 会话管理
	RevokeSession(id uint, reason string) error
	RevokeAllUserSessions(userID uint, reason string) error
	RevokeExpiredSessions() error
	UpdateLastUsed(id uint) error

	// 统计操作
	CountByUserID(userID uint) (int64, error)
	CountActiveByUserID(userID uint) (int64, error)
}

// sessionRepository 会话仓库实现
type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 创建会话仓库实例
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// Create 创建会话
func (r *sessionRepository) Create(session *entity.Session) error {
	return r.db.Create(session).Error
}

// GetByID 根据ID获取会话
func (r *sessionRepository) GetByID(id uint) (*entity.Session, error) {
	var session entity.Session
	if err := r.db.First(&session, id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// GetByAccessToken 根据访问令牌获取会话
func (r *sessionRepository) GetByAccessToken(accessToken string) (*entity.Session, error) {
	var session entity.Session
	if err := r.db.Where("access_token = ?", accessToken).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// GetByRefreshToken 根据刷新令牌获取会话
func (r *sessionRepository) GetByRefreshToken(refreshToken string) (*entity.Session, error) {
	var session entity.Session
	if err := r.db.Where("refresh_token = ?", refreshToken).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// Update 更新会话
func (r *sessionRepository) Update(session *entity.Session) error {
	return r.db.Save(session).Error
}

// Delete 删除会话（软删除）
func (r *sessionRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Session{}, id).Error
}

// GetByUserID 根据用户ID获取会话列表
func (r *sessionRepository) GetByUserID(userID uint, limit, offset int) ([]entity.Session, error) {
	var sessions []entity.Session
	query := r.db.Where("user_id = ?", userID).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetActiveByUserID 获取用户的活跃会话
func (r *sessionRepository) GetActiveByUserID(userID uint) ([]entity.Session, error) {
	var sessions []entity.Session
	now := time.Now()
	query := r.db.Where("user_id = ? AND is_active = ? AND revoked_at IS NULL AND access_token_expires_at > ?",
		userID, true, now)
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetExpiredSessions 获取所有过期的会话
func (r *sessionRepository) GetExpiredSessions() ([]entity.Session, error) {
	var sessions []entity.Session
	now := time.Now()
	query := r.db.Where("access_token_expires_at < ? OR refresh_token_expires_at < ?", now, now)
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// RevokeSession 撤销会话
func (r *sessionRepository) RevokeSession(id uint, reason string) error {
	now := time.Now()
	return r.db.Model(&entity.Session{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":    false,
			"revoked_at":   &now,
			"revoke_reason": reason,
		}).Error
}

// RevokeAllUserSessions 撤销用户的所有会话
func (r *sessionRepository) RevokeAllUserSessions(userID uint, reason string) error {
	now := time.Now()
	return r.db.Model(&entity.Session{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Updates(map[string]interface{}{
			"is_active":     false,
			"revoked_at":    &now,
			"revoke_reason": reason,
		}).Error
}

// RevokeExpiredSessions 撤销所有过期的会话
func (r *sessionRepository) RevokeExpiredSessions() error {
	now := time.Now()
	return r.db.Model(&entity.Session{}).
		Where("(access_token_expires_at < ? OR refresh_token_expires_at < ?) AND is_active = ?",
			now, now, true).
		Updates(map[string]interface{}{
			"is_active":     false,
			"revoked_at":    &now,
			"revoke_reason": "自动撤销：会话已过期",
		}).Error
}

// UpdateLastUsed 更新会话的最后使用时间
func (r *sessionRepository) UpdateLastUsed(id uint) error {
	return r.db.Model(&entity.Session{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

// CountByUserID 统计用户的会话数量
func (r *sessionRepository) CountByUserID(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Session{}).
		Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveByUserID 统计用户的活跃会话数量
func (r *sessionRepository) CountActiveByUserID(userID uint) (int64, error) {
	var count int64
	now := time.Now()
	if err := r.db.Model(&entity.Session{}).
		Where("user_id = ? AND is_active = ? AND revoked_at IS NULL AND access_token_expires_at > ?",
			userID, true, now).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
