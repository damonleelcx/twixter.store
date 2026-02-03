package config

import (
	"errors"
	"log"

	"backend/entity"
	"gorm.io/gorm"
)

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

	log.Println("Permissions initialization completed")
}
