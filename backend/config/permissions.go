package config

import (
	"errors"
	"log"

	"backend/entity"
	"gorm.io/gorm"
)

// grantCanUseBotToDarkAndAdmin 为所有已存在的 dark / admin 用户授予 can_use_bot（仅 dark 与 admin 可拥有）
func grantCanUseBotToDarkAndAdmin(db *gorm.DB) {
	var perm entity.Permission
	if err := db.Where("name = ?", "can_use_bot").First(&perm).Error; err != nil || perm.ID == 0 {
		return
	}
	accountTypes := []string{string(entity.AccountTypeDark), string(entity.AccountTypeAdmin)}
	var ids0 []uint
	if err := db.Model(&entity.UserShard0{}).Where("account_type IN ?", accountTypes).Pluck("id", &ids0).Error; err != nil {
		log.Printf("grantCanUseBotToDarkAndAdmin shard0: %v", err)
		return
	}
	var ids1 []uint
	if err := db.Model(&entity.UserShard1{}).Where("account_type IN ?", accountTypes).Pluck("id", &ids1).Error; err != nil {
		log.Printf("grantCanUseBotToDarkAndAdmin shard1: %v", err)
		return
	}
	for _, userID := range ids0 {
		var ex entity.UserPermission
		if err := db.Where("user_id = ? AND permission_id = ?", userID, perm.ID).First(&ex).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		up := &entity.UserPermission{UserID: userID, PermissionID: perm.ID, ShardNumber: 0}
		if err := db.Create(up).Error; err != nil {
			log.Printf("grant can_use_bot to user %d: %v", userID, err)
		}
	}
	for _, userID := range ids1 {
		var ex entity.UserPermission
		if err := db.Where("user_id = ? AND permission_id = ?", userID, perm.ID).First(&ex).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		up := &entity.UserPermission{UserID: userID, PermissionID: perm.ID, ShardNumber: 1}
		if err := db.Create(up).Error; err != nil {
			log.Printf("grant can_use_bot to user %d: %v", userID, err)
		}
	}
	log.Println("Granted can_use_bot to existing dark/admin users")
}

// InitPermissions 初始化系统权限
func InitPermissions(db *gorm.DB) {
	permissions := []entity.Permission{
		{
			Name:        "can_upload_content",
			Description: "允许上传内容",
			Resource:    "content",
			Action:      entity.ActionCreate,
			IsSystem:    true,
		},
		{
			Name:        "can_edit_content",
			Description: "允许编辑内容（仅 admin 拥有）",
			Resource:    "content",
			Action:      entity.ActionEdit,
			IsSystem:    true,
		},
		{
			Name:        "can_delete_content",
			Description: "允许删除内容（仅 admin 拥有）",
			Resource:    "content",
			Action:      entity.ActionDelete,
			IsSystem:    true,
		},
		{
			Name:        "can_view_analytics",
			Description: "允许查看分析数据",
			Resource:    "analytics",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
		{
			Name:        "can_view_all",
			Description: "允许查看全部内容（仅 admin 拥有，无需购买或会员）",
			Resource:    "content",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
		{
			Name:        "can_view_nsfw",
			Description: "允许查看NSFW内容",
			Resource:    "nsfw",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
		{
			Name:        "can_search_tags",
			Description: "允许搜索标签",
			Resource:    "tags",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
		{
			Name:        "can_use_bot",
			Description: "允许使用 Agent 聊天（仅 dark / admin 账户类型）",
			Resource:    "bot",
			Action:      entity.ActionView,
			IsSystem:    true,
		},
	}

	for _, perm := range permissions {
		// 使用 FirstOrCreate 来避免重复创建
		var existing entity.Permission
		result := db.Where("name = ? OR (resource = ? AND action = ?)",
			perm.Name, perm.Resource, perm.Action).First(&existing)

		if result.Error != nil {
			// 如果权限不存在，则创建
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := db.Create(&perm).Error; err != nil {
					log.Printf("Failed to create permission %s: %v", perm.Name, err)
				} else {
					log.Printf("Created permission: %s", perm.Name)
				}
			} else {
				log.Printf("Error checking permission %s: %v", perm.Name, result.Error)
			}
		} else {
			// 权限已存在，更新描述和系统标志（如果需要）
			if existing.Description != perm.Description || existing.IsSystem != perm.IsSystem {
				existing.Description = perm.Description
				existing.IsSystem = perm.IsSystem
				if err := db.Save(&existing).Error; err != nil {
					log.Printf("Failed to update permission %s: %v", perm.Name, err)
				} else {
					log.Printf("Updated permission: %s", perm.Name)
				}
			}
		}
	}

	grantCanUseBotToDarkAndAdmin(db)
	log.Println("Permissions initialization completed")
}
