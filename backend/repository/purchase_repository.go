package repository

import (
	"backend/entity"
	"errors"

	"gorm.io/gorm"
)

// PurchaseRepository 购买仓库接口
type PurchaseRepository interface {
	// 基础 CRUD 操作
	Create(purchase *entity.Purchase) error
	GetByID(id uint) (*entity.Purchase, error)
	Update(purchase *entity.Purchase) error
	Delete(id uint) error

	// 分片表操作
	CreateInShard(userID uint, purchase *entity.PurchaseBase) error
	GetByIDFromShard(userID uint, id uint) (*entity.PurchaseBase, error)
	UpdateInShard(userID uint, purchase *entity.PurchaseBase) error
	DeleteFromShard(userID uint, id uint) error

	// 查询操作
	GetByUserID(userID uint, limit, offset int) ([]entity.PurchaseBase, error)
	GetByType(userID uint, purchaseType entity.PurchaseType, limit, offset int) ([]entity.PurchaseBase, error)
	GetByStatus(userID uint, status entity.PurchaseStatus, limit, offset int) ([]entity.PurchaseBase, error)
	GetByContentID(contentID uint, limit, offset int) ([]entity.PurchaseBase, error)
	GetByTransactionID(transactionID string) (*entity.PurchaseBase, error)
	GetByGatewayOrderID(gatewayOrderID string) (*entity.PurchaseBase, error)

	// 统计操作
	CountByUserID(userID uint) (int64, error)
	CountByType(userID uint, purchaseType entity.PurchaseType) (int64, error)
	CountByStatus(userID uint, status entity.PurchaseStatus) (int64, error)
	GetTotalRevenue(purchaseType entity.PurchaseType) (float64, error)
}

// purchaseRepository 购买仓库实现
type purchaseRepository struct {
	db *gorm.DB
}

// NewPurchaseRepository 创建购买仓库实例
func NewPurchaseRepository(db *gorm.DB) PurchaseRepository {
	return &purchaseRepository{db: db}
}

// Create 创建购买记录（使用默认表）
func (r *purchaseRepository) Create(purchase *entity.Purchase) error {
	return r.db.Create(purchase).Error
}

// GetByID 根据ID获取购买记录（使用默认表）
func (r *purchaseRepository) GetByID(id uint) (*entity.Purchase, error) {
	var purchase entity.Purchase
	if err := r.db.First(&purchase, id).Error; err != nil {
		return nil, err
	}
	return &purchase, nil
}

// Update 更新购买记录（使用默认表）
func (r *purchaseRepository) Update(purchase *entity.Purchase) error {
	return r.db.Save(purchase).Error
}

// Delete 删除购买记录（使用默认表，软删除）
func (r *purchaseRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Purchase{}, id).Error
}

// CreateInShard 在分片表中创建购买记录
func (r *purchaseRepository) CreateInShard(userID uint, purchase *entity.PurchaseBase) error {
	shardNum := entity.GetPurchaseShardNumber(userID)
	if shardNum == 0 {
		shardPurchase := entity.PurchaseShard0{PurchaseBase: *purchase}
		return r.db.Create(&shardPurchase).Error
	}
	shardPurchase := entity.PurchaseShard1{PurchaseBase: *purchase}
	return r.db.Create(&shardPurchase).Error
}

// GetByIDFromShard 从分片表中根据ID获取购买记录
func (r *purchaseRepository) GetByIDFromShard(userID uint, id uint) (*entity.PurchaseBase, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	if shardNum == 0 {
		var purchase entity.PurchaseShard0
		if err := r.db.First(&purchase, id).Error; err != nil {
			return nil, err
		}
		return &purchase.PurchaseBase, nil
	}
	var purchase entity.PurchaseShard1
	if err := r.db.First(&purchase, id).Error; err != nil {
		return nil, err
	}
	return &purchase.PurchaseBase, nil
}

// UpdateInShard 在分片表中更新购买记录
func (r *purchaseRepository) UpdateInShard(userID uint, purchase *entity.PurchaseBase) error {
	shardNum := entity.GetPurchaseShardNumber(userID)
	if shardNum == 0 {
		shardPurchase := entity.PurchaseShard0{PurchaseBase: *purchase}
		return r.db.Save(&shardPurchase).Error
	}
	shardPurchase := entity.PurchaseShard1{PurchaseBase: *purchase}
	return r.db.Save(&shardPurchase).Error
}

// DeleteFromShard 从分片表中删除购买记录（软删除）
func (r *purchaseRepository) DeleteFromShard(userID uint, id uint) error {
	shardNum := entity.GetPurchaseShardNumber(userID)
	if shardNum == 0 {
		return r.db.Delete(&entity.PurchaseShard0{}, id).Error
	}
	return r.db.Delete(&entity.PurchaseShard1{}, id).Error
}

// GetByUserID 根据用户ID获取购买记录列表（从分片表）
func (r *purchaseRepository) GetByUserID(userID uint, limit, offset int) ([]entity.PurchaseBase, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	var purchases []entity.PurchaseBase

	if shardNum == 0 {
		var shardPurchases []entity.PurchaseShard0
		query := r.db.Where("user_id = ?", userID).
			Limit(limit).Offset(offset).Order("created_at DESC")
		if err := query.Find(&shardPurchases).Error; err != nil {
			return nil, err
		}
		for _, p := range shardPurchases {
			purchases = append(purchases, p.PurchaseBase)
		}
	} else {
		var shardPurchases []entity.PurchaseShard1
		query := r.db.Where("user_id = ?", userID).
			Limit(limit).Offset(offset).Order("created_at DESC")
		if err := query.Find(&shardPurchases).Error; err != nil {
			return nil, err
		}
		for _, p := range shardPurchases {
			purchases = append(purchases, p.PurchaseBase)
		}
	}

	return purchases, nil
}

// GetByType 根据类型获取购买记录列表（从分片表）
func (r *purchaseRepository) GetByType(userID uint, purchaseType entity.PurchaseType, limit, offset int) ([]entity.PurchaseBase, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	var purchases []entity.PurchaseBase

	if shardNum == 0 {
		var shardPurchases []entity.PurchaseShard0
		query := r.db.Where("user_id = ? AND purchase_type = ?", userID, purchaseType).
			Limit(limit).Offset(offset).Order("created_at DESC")
		if err := query.Find(&shardPurchases).Error; err != nil {
			return nil, err
		}
		for _, p := range shardPurchases {
			purchases = append(purchases, p.PurchaseBase)
		}
	} else {
		var shardPurchases []entity.PurchaseShard1
		query := r.db.Where("user_id = ? AND purchase_type = ?", userID, purchaseType).
			Limit(limit).Offset(offset).Order("created_at DESC")
		if err := query.Find(&shardPurchases).Error; err != nil {
			return nil, err
		}
		for _, p := range shardPurchases {
			purchases = append(purchases, p.PurchaseBase)
		}
	}

	return purchases, nil
}

// GetByStatus 根据状态获取购买记录列表（从分片表）
func (r *purchaseRepository) GetByStatus(userID uint, status entity.PurchaseStatus, limit, offset int) ([]entity.PurchaseBase, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	var purchases []entity.PurchaseBase

	if shardNum == 0 {
		var shardPurchases []entity.PurchaseShard0
		query := r.db.Where("user_id = ? AND status = ?", userID, status).
			Limit(limit).Offset(offset).Order("created_at DESC")
		if err := query.Find(&shardPurchases).Error; err != nil {
			return nil, err
		}
		for _, p := range shardPurchases {
			purchases = append(purchases, p.PurchaseBase)
		}
	} else {
		var shardPurchases []entity.PurchaseShard1
		query := r.db.Where("user_id = ? AND status = ?", userID, status).
			Limit(limit).Offset(offset).Order("created_at DESC")
		if err := query.Find(&shardPurchases).Error; err != nil {
			return nil, err
		}
		for _, p := range shardPurchases {
			purchases = append(purchases, p.PurchaseBase)
		}
	}

	return purchases, nil
}

// GetByContentID 根据内容ID获取购买记录（需要搜索所有分片）
func (r *purchaseRepository) GetByContentID(contentID uint, limit, offset int) ([]entity.PurchaseBase, error) {
	var purchases []entity.PurchaseBase

	// 搜索分片0
	var shard0Purchases []entity.PurchaseShard0
	query0 := r.db.Where("content_id = ?", contentID).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query0.Find(&shard0Purchases).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	for _, p := range shard0Purchases {
		purchases = append(purchases, p.PurchaseBase)
	}

	// 搜索分片1
	var shard1Purchases []entity.PurchaseShard1
	query1 := r.db.Where("content_id = ?", contentID).
		Limit(limit).Offset(offset).Order("created_at DESC")
	if err := query1.Find(&shard1Purchases).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	for _, p := range shard1Purchases {
		purchases = append(purchases, p.PurchaseBase)
	}

	return purchases, nil
}

// GetByTransactionID 根据交易ID获取购买记录（需要搜索所有分片）
func (r *purchaseRepository) GetByTransactionID(transactionID string) (*entity.PurchaseBase, error) {
	// 先搜索分片0
	var purchase0 entity.PurchaseShard0
	if err := r.db.Where("transaction_id = ?", transactionID).First(&purchase0).Error; err == nil {
		return &purchase0.PurchaseBase, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 再搜索分片1
	var purchase1 entity.PurchaseShard1
	if err := r.db.Where("transaction_id = ?", transactionID).First(&purchase1).Error; err != nil {
		return nil, err
	}
	return &purchase1.PurchaseBase, nil
}

// GetByGatewayOrderID 根据支付网关订单ID获取购买记录（需要搜索所有分片）
func (r *purchaseRepository) GetByGatewayOrderID(gatewayOrderID string) (*entity.PurchaseBase, error) {
	// 先搜索分片0
	var purchase0 entity.PurchaseShard0
	if err := r.db.Where("gateway_order_id = ?", gatewayOrderID).First(&purchase0).Error; err == nil {
		return &purchase0.PurchaseBase, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 再搜索分片1
	var purchase1 entity.PurchaseShard1
	if err := r.db.Where("gateway_order_id = ?", gatewayOrderID).First(&purchase1).Error; err != nil {
		return nil, err
	}
	return &purchase1.PurchaseBase, nil
}

// CountByUserID 统计用户的购买记录数量（从分片表）
func (r *purchaseRepository) CountByUserID(userID uint) (int64, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	var count int64

	if shardNum == 0 {
		if err := r.db.Model(&entity.PurchaseShard0{}).
			Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return 0, err
		}
	} else {
		if err := r.db.Model(&entity.PurchaseShard1{}).
			Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return 0, err
		}
	}

	return count, nil
}

// CountByType 统计用户指定类型的购买记录数量（从分片表）
func (r *purchaseRepository) CountByType(userID uint, purchaseType entity.PurchaseType) (int64, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	var count int64

	if shardNum == 0 {
		if err := r.db.Model(&entity.PurchaseShard0{}).
			Where("user_id = ? AND purchase_type = ?", userID, purchaseType).Count(&count).Error; err != nil {
			return 0, err
		}
	} else {
		if err := r.db.Model(&entity.PurchaseShard1{}).
			Where("user_id = ? AND purchase_type = ?", userID, purchaseType).Count(&count).Error; err != nil {
			return 0, err
		}
	}

	return count, nil
}

// CountByStatus 统计用户指定状态的购买记录数量（从分片表）
func (r *purchaseRepository) CountByStatus(userID uint, status entity.PurchaseStatus) (int64, error) {
	shardNum := entity.GetPurchaseShardNumber(userID)
	var count int64

	if shardNum == 0 {
		if err := r.db.Model(&entity.PurchaseShard0{}).
			Where("user_id = ? AND status = ?", userID, status).Count(&count).Error; err != nil {
			return 0, err
		}
	} else {
		if err := r.db.Model(&entity.PurchaseShard1{}).
			Where("user_id = ? AND status = ?", userID, status).Count(&count).Error; err != nil {
			return 0, err
		}
	}

	return count, nil
}

// GetTotalRevenue 获取总收入（需要搜索所有分片）
func (r *purchaseRepository) GetTotalRevenue(purchaseType entity.PurchaseType) (float64, error) {
	var total0, total1 float64

	// 统计分片0
	if err := r.db.Model(&entity.PurchaseShard0{}).
		Where("purchase_type = ? AND status = ?", purchaseType, entity.PurchaseStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total0).Error; err != nil {
		return 0, err
	}

	// 统计分片1
	if err := r.db.Model(&entity.PurchaseShard1{}).
		Where("purchase_type = ? AND status = ?", purchaseType, entity.PurchaseStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total1).Error; err != nil {
		return 0, err
	}

	return total0 + total1, nil
}
