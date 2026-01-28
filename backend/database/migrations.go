package database

import (
	"log"

	"backend/entity"
	"gorm.io/gorm"
)

// RunMigrations 运行数据库迁移和创建索引
func RunMigrations(db *gorm.DB) error {
	// 自动迁移数据库表（包括分片表）
	if err := db.AutoMigrate(
		&entity.UserShard0{},
		&entity.UserShard1{},
		&entity.Session{},
		&entity.Permission{},
		&entity.UserPermission{},
		&entity.Content{},
		&entity.ContentFile{},
		&entity.Tag{},
		&entity.ContentTag{},
		&entity.PurchaseShard0{},
		&entity.PurchaseShard1{},
		&entity.PurchaseShard2{},
		&entity.PurchaseShard3{},
		&entity.Wallet{},
		&entity.Analytics{},
	); err != nil {
		return err
	}

	// 创建所有索引
	if err := createIndexes(db); err != nil {
		return err
	}

	log.Println("Database migration completed successfully (users_shard_0, users_shard_1, sessions, permissions, user_permissions, contents, tags, content_tags, purchases_shard_0, purchases_shard_1, purchases_shard_2, purchases_shard_3, wallets, analytics)")
	return nil
}

// createIndexes 创建所有数据库索引
func createIndexes(db *gorm.DB) error {
	indexes := []struct {
		name string
		sql  string
	}{
		// 权限相关索引
		{
			name: "idx_permissions_resource_action_unique",
			sql: `CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_resource_action_unique 
				ON permissions(resource, action) 
				WHERE deleted_at IS NULL;`,
		},
		{
			name: "idx_user_permissions_unique",
			sql: `CREATE UNIQUE INDEX IF NOT EXISTS idx_user_permissions_unique 
				ON user_permissions(user_id, permission_id) 
				WHERE deleted_at IS NULL;`,
		},
		{
			name: "idx_user_permissions_user_shard",
			sql: `CREATE INDEX IF NOT EXISTS idx_user_permissions_user_shard 
				ON user_permissions(user_id, shard_number, deleted_at);`,
		},
		// 会话相关索引
		{
			name: "idx_sessions_user_shard",
			sql: `CREATE INDEX IF NOT EXISTS idx_sessions_user_shard 
				ON sessions(user_id, shard_number, deleted_at);`,
		},
		// 内容相关索引
		{
			name: "idx_contents_uploaded_by_shard",
			sql: `CREATE INDEX IF NOT EXISTS idx_contents_uploaded_by_shard 
				ON contents(uploaded_by, shard_number, deleted_at);`,
		},
		{
			name: "idx_contents_type_status",
			sql: `CREATE INDEX IF NOT EXISTS idx_contents_type_status 
				ON contents(type, status, deleted_at);`,
		},
		{
			name: "idx_contents_public_created",
			sql: `CREATE INDEX IF NOT EXISTS idx_contents_public_created 
				ON contents(is_public, created_at DESC) 
				WHERE deleted_at IS NULL;`,
		},
		// 内容标签相关索引
		{
			name: "idx_content_tags_unique",
			sql: `CREATE UNIQUE INDEX IF NOT EXISTS idx_content_tags_unique 
				ON content_tags(content_id, tag_id) 
				WHERE deleted_at IS NULL;`,
		},
		{
			name: "idx_content_tags_content",
			sql: `CREATE INDEX IF NOT EXISTS idx_content_tags_content 
				ON content_tags(content_id, deleted_at);`,
		},
		{
			name: "idx_content_tags_tag",
			sql: `CREATE INDEX IF NOT EXISTS idx_content_tags_tag 
				ON content_tags(tag_id, deleted_at);`,
		},
		// 购买记录相关索引（分片0-3）
		{
			name: "idx_purchases_shard_0_user",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_user 
				ON purchases_shard_0(user_id, purchase_type, status, deleted_at);`,
		},
		{
			name: "idx_purchases_shard_1_user",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_user 
				ON purchases_shard_1(user_id, purchase_type, status, deleted_at);`,
		},
		{
			name: "idx_purchases_shard_2_user",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_2_user 
				ON purchases_shard_2(user_id, purchase_type, status, deleted_at);`,
		},
		{
			name: "idx_purchases_shard_3_user",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_3_user 
				ON purchases_shard_3(user_id, purchase_type, status, deleted_at);`,
		},
		{
			name: "idx_purchases_shard_0_content",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_content 
				ON purchases_shard_0(content_id, deleted_at) 
				WHERE content_id IS NOT NULL;`,
		},
		{
			name: "idx_purchases_shard_1_content",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_content 
				ON purchases_shard_1(content_id, deleted_at) 
				WHERE content_id IS NOT NULL;`,
		},
		{
			name: "idx_purchases_shard_2_content",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_2_content 
				ON purchases_shard_2(content_id, deleted_at) 
				WHERE content_id IS NOT NULL;`,
		},
		{
			name: "idx_purchases_shard_3_content",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_3_content 
				ON purchases_shard_3(content_id, deleted_at) 
				WHERE content_id IS NOT NULL;`,
		},
		{
			name: "idx_purchases_shard_0_transaction",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_transaction 
				ON purchases_shard_0(transaction_id, deleted_at) 
				WHERE transaction_id != '';`,
		},
		{
			name: "idx_purchases_shard_1_transaction",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_transaction 
				ON purchases_shard_1(transaction_id, deleted_at) 
				WHERE transaction_id != '';`,
		},
		{
			name: "idx_purchases_shard_2_transaction",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_2_transaction 
				ON purchases_shard_2(transaction_id, deleted_at) 
				WHERE transaction_id != '';`,
		},
		{
			name: "idx_purchases_shard_3_transaction",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_3_transaction 
				ON purchases_shard_3(transaction_id, deleted_at) 
				WHERE transaction_id != '';`,
		},
		{
			name: "idx_purchases_shard_0_gateway_order",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_gateway_order 
				ON purchases_shard_0(gateway_order_id, deleted_at) 
				WHERE gateway_order_id != '';`,
		},
		{
			name: "idx_purchases_shard_1_gateway_order",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_gateway_order 
				ON purchases_shard_1(gateway_order_id, deleted_at) 
				WHERE gateway_order_id != '';`,
		},
		{
			name: "idx_purchases_shard_2_gateway_order",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_2_gateway_order 
				ON purchases_shard_2(gateway_order_id, deleted_at) 
				WHERE gateway_order_id != '';`,
		},
		{
			name: "idx_purchases_shard_3_gateway_order",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_3_gateway_order 
				ON purchases_shard_3(gateway_order_id, deleted_at) 
				WHERE gateway_order_id != '';`,
		},
		{
			name: "idx_purchases_shard_0_gateway_status",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_0_gateway_status 
				ON purchases_shard_0(payment_gateway, status, deleted_at) 
				WHERE payment_gateway != '';`,
		},
		{
			name: "idx_purchases_shard_1_gateway_status",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_1_gateway_status 
				ON purchases_shard_1(payment_gateway, status, deleted_at) 
				WHERE payment_gateway != '';`,
		},
		{
			name: "idx_purchases_shard_2_gateway_status",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_2_gateway_status 
				ON purchases_shard_2(payment_gateway, status, deleted_at) 
				WHERE payment_gateway != '';`,
		},
		{
			name: "idx_purchases_shard_3_gateway_status",
			sql: `CREATE INDEX IF NOT EXISTS idx_purchases_shard_3_gateway_status 
				ON purchases_shard_3(payment_gateway, status, deleted_at) 
				WHERE payment_gateway != '';`,
		},
		// 钱包相关索引
		{
			name: "idx_wallets_user_shard",
			sql: `CREATE INDEX IF NOT EXISTS idx_wallets_user_shard 
				ON wallets(user_id, shard_number, deleted_at);`,
		},
		{
			name: "idx_wallets_user_unique",
			sql: `CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_unique 
				ON wallets(user_id) 
				WHERE deleted_at IS NULL;`,
		},
		// 分析数据相关索引
		{
			name: "idx_analytics_content_date_unique",
			sql: `CREATE UNIQUE INDEX IF NOT EXISTS idx_analytics_content_date_unique 
				ON analytics(content_id, date) 
				WHERE deleted_at IS NULL;`,
		},
		{
			name: "idx_analytics_content_date",
			sql: `CREATE INDEX IF NOT EXISTS idx_analytics_content_date 
				ON analytics(content_id, date DESC, deleted_at);`,
		},
		{
			name: "idx_analytics_date",
			sql: `CREATE INDEX IF NOT EXISTS idx_analytics_date 
				ON analytics(date DESC, deleted_at);`,
		},
		{
			name: "idx_analytics_views",
			sql: `CREATE INDEX IF NOT EXISTS idx_analytics_views 
				ON analytics(views DESC, date DESC) 
				WHERE deleted_at IS NULL;`,
		},
	}

	for _, idx := range indexes {
		if err := db.Exec(idx.sql).Error; err != nil {
			log.Printf("Warning: Failed to create index %s: %v", idx.name, err)
		}
	}

	return nil
}
