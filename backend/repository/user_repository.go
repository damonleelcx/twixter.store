package repository

import (
	"backend/entity"
	"errors"

	"gorm.io/gorm"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	// 基础 CRUD 操作
	Create(user *entity.User) error
	GetByID(id uint) (*entity.User, error)
	GetByEmail(email string) (*entity.User, error)
	Update(user *entity.User) error
	Delete(id uint) error

	// 分片表操作
	CreateInShard(userID uint, user *entity.UserBase) error
	GetByIDFromShard(userID uint, id uint) (*entity.UserBase, error)
	GetByEmailFromShard(email string) (*entity.UserBase, error)
	UpdateInShard(userID uint, user *entity.UserBase) error
	DeleteFromShard(userID uint, id uint) error

	// 查询操作
	List(limit, offset int) ([]entity.User, error)
	Count() (int64, error)
	GetByAccountType(accountType entity.AccountType, limit, offset int) ([]entity.User, error)
	GetByReferralCode(referralCode string) (*entity.UserBase, error) // 根据推荐码查找用户
	CountByAccountTypeInShards(accountType entity.AccountType) (int64, error) // 统计分片中某账户类型的数量
}

// userRepository 用户仓库实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户（使用默认表）
func (r *userRepository) Create(user *entity.User) error {
	return r.db.Create(user).Error
}

// GetByID 根据ID获取用户（使用默认表）
func (r *userRepository) GetByID(id uint) (*entity.User, error) {
	var user entity.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户（使用默认表）
func (r *userRepository) GetByEmail(email string) (*entity.User, error) {
	var user entity.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户（使用默认表）
func (r *userRepository) Update(user *entity.User) error {
	return r.db.Save(user).Error
}

// Delete 删除用户（使用默认表，软删除）
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&entity.User{}, id).Error
}

// CreateInShard 在分片表中创建用户
func (r *userRepository) CreateInShard(userID uint, user *entity.UserBase) error {
	shardNum := entity.GetShardNumber(userID)
	if shardNum == 0 {
		shardUser := entity.UserShard0{UserBase: *user}
		return r.db.Create(&shardUser).Error
	}
	shardUser := entity.UserShard1{UserBase: *user}
	return r.db.Create(&shardUser).Error
}

// GetByIDFromShard 从分片表中根据ID获取用户
func (r *userRepository) GetByIDFromShard(userID uint, id uint) (*entity.UserBase, error) {
	shardNum := entity.GetShardNumber(userID)
	if shardNum == 0 {
		var user entity.UserShard0
		if err := r.db.First(&user, id).Error; err != nil {
			return nil, err
		}
		return &user.UserBase, nil
	}
	var user entity.UserShard1
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user.UserBase, nil
}

// GetByEmailFromShard 从分片表中根据邮箱获取用户（需要搜索所有分片）
func (r *userRepository) GetByEmailFromShard(email string) (*entity.UserBase, error) {
	// 先搜索分片0
	var user0 entity.UserShard0
	if err := r.db.Where("email = ?", email).First(&user0).Error; err == nil {
		return &user0.UserBase, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 再搜索分片1
	var user1 entity.UserShard1
	if err := r.db.Where("email = ?", email).First(&user1).Error; err != nil {
		return nil, err
	}
	return &user1.UserBase, nil
}

// UpdateInShard 在分片表中更新用户
func (r *userRepository) UpdateInShard(userID uint, user *entity.UserBase) error {
	shardNum := entity.GetShardNumber(userID)
	if shardNum == 0 {
		shardUser := entity.UserShard0{UserBase: *user}
		return r.db.Save(&shardUser).Error
	}
	shardUser := entity.UserShard1{UserBase: *user}
	return r.db.Save(&shardUser).Error
}

// DeleteFromShard 从分片表中删除用户（软删除）
func (r *userRepository) DeleteFromShard(userID uint, id uint) error {
	shardNum := entity.GetShardNumber(userID)
	if shardNum == 0 {
		return r.db.Delete(&entity.UserShard0{}, id).Error
	}
	return r.db.Delete(&entity.UserShard1{}, id).Error
}

// List 列出用户（使用默认表）
func (r *userRepository) List(limit, offset int) ([]entity.User, error) {
	var users []entity.User
	query := r.db.Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Count 统计用户数量（使用默认表）
func (r *userRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&entity.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetByAccountType 根据账户类型获取用户列表（使用默认表）
func (r *userRepository) GetByAccountType(accountType entity.AccountType, limit, offset int) ([]entity.User, error) {
	var users []entity.User
	query := r.db.Where("account_type = ?", accountType).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetByReferralCode 根据推荐码查找用户（需要搜索所有分片）
func (r *userRepository) GetByReferralCode(referralCode string) (*entity.UserBase, error) {
	// 先搜索分片0
	var user0 entity.UserShard0
	if err := r.db.Where("referral_code = ?", referralCode).First(&user0).Error; err == nil {
		return &user0.UserBase, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 再搜索分片1
	var user1 entity.UserShard1
	if err := r.db.Where("referral_code = ?", referralCode).First(&user1).Error; err != nil {
		return nil, err
	}
	return &user1.UserBase, nil
}

// CountByAccountTypeInShards 统计分片中某账户类型的用户数量（users 表在注册后会删除记录，以分片为准）
func (r *userRepository) CountByAccountTypeInShards(accountType entity.AccountType) (int64, error) {
	var c0, c1 int64
	if err := r.db.Model(&entity.UserShard0{}).Where("account_type = ?", accountType).Count(&c0).Error; err != nil {
		return 0, err
	}
	if err := r.db.Model(&entity.UserShard1{}).Where("account_type = ?", accountType).Count(&c1).Error; err != nil {
		return 0, err
	}
	return c0 + c1, nil
}
