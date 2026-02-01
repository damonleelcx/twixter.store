package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// WalletRepository 钱包仓库接口
type WalletRepository interface {
	// 基础 CRUD 操作
	Create(wallet *entity.Wallet) error
	GetByID(id uint) (*entity.Wallet, error)
	GetByUserID(userID uint) (*entity.Wallet, error)
	Update(wallet *entity.Wallet) error
	Delete(id uint) error

	// 钱包操作
	AddCredits(userID uint, amount int64) error
	SpendCredits(userID uint, amount int64) error
	GetBalance(userID uint) (int64, error)
	HasEnoughCredits(userID uint, amount int64) (bool, error)

	// 查询操作
	List(limit, offset int) ([]entity.Wallet, error)
	GetTopWallets(limit int) ([]entity.Wallet, error)

	// 统计操作
	Count() (int64, error)
	GetTotalBalance() (int64, error)
}

// walletRepository 钱包仓库实现
type walletRepository struct {
	db *gorm.DB
}

// NewWalletRepository 创建钱包仓库实例
func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

// Create 创建钱包
func (r *walletRepository) Create(wallet *entity.Wallet) error {
	return r.db.Create(wallet).Error
}

// GetByID 根据ID获取钱包
func (r *walletRepository) GetByID(id uint) (*entity.Wallet, error) {
	var wallet entity.Wallet
	if err := r.db.First(&wallet, id).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

// GetByUserID 根据用户ID获取钱包
func (r *walletRepository) GetByUserID(userID uint) (*entity.Wallet, error) {
	var wallet entity.Wallet
	if err := r.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

// Update 更新钱包
func (r *walletRepository) Update(wallet *entity.Wallet) error {
	return r.db.Save(wallet).Error
}

// Delete 删除钱包（软删除）
func (r *walletRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Wallet{}, id).Error
}

// AddCredits 增加积分（使用事务确保原子性）
func (r *walletRepository) AddCredits(userID uint, amount int64) error {
	if amount <= 0 {
		return nil // 无效金额，直接返回
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var wallet entity.Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}

		wallet.Balance += amount
		wallet.TotalEarned += amount

		return tx.Save(&wallet).Error
	})
}

// SpendCredits 消费积分（使用事务确保原子性）
func (r *walletRepository) SpendCredits(userID uint, amount int64) error {
	if amount <= 0 {
		return nil // 无效金额，直接返回
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var wallet entity.Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}

		if wallet.Balance < amount {
			return gorm.ErrRecordNotFound // 余额不足
		}

		wallet.Balance -= amount
		wallet.TotalSpent += amount

		return tx.Save(&wallet).Error
	})
}

// GetBalance 获取用户余额
func (r *walletRepository) GetBalance(userID uint) (int64, error) {
	var wallet entity.Wallet
	if err := r.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return 0, err
	}
	return wallet.Balance, nil
}

// HasEnoughCredits 检查用户是否有足够的积分
func (r *walletRepository) HasEnoughCredits(userID uint, amount int64) (bool, error) {
	balance, err := r.GetBalance(userID)
	if err != nil {
		return false, err
	}
	return balance >= amount, nil
}

// List 列出钱包
func (r *walletRepository) List(limit, offset int) ([]entity.Wallet, error) {
	var wallets []entity.Wallet
	query := r.db.Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query.Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

// GetTopWallets 获取余额最高的钱包
func (r *walletRepository) GetTopWallets(limit int) ([]entity.Wallet, error) {
	var wallets []entity.Wallet
	query := r.db.Order("balance DESC").Limit(limit)
	if err := query.Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

// Count 统计钱包数量
func (r *walletRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&entity.Wallet{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetTotalBalance 获取所有钱包的总余额
func (r *walletRepository) GetTotalBalance() (int64, error) {
	var total int64
	if err := r.db.Model(&entity.Wallet{}).
		Select("COALESCE(SUM(balance), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
