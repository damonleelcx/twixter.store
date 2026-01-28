package middleware

import (
	"backend/entity"
	"backend/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequirePermission 要求用户拥有指定权限的中间件
// 使用权限名称进行检查
func RequirePermission(userPermissionRepo repository.UserPermissionRepository, permissionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		user, exists := GetUserFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有指定权限
		hasPermission, err := userPermissionRepo.HasPermission(user.ID, permissionName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permission",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":               "Insufficient permissions",
				"required_permission": permissionName,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermissionByResourceAction 要求用户拥有指定资源和操作权限的中间件
// 使用资源和操作类型进行检查
func RequirePermissionByResourceAction(userPermissionRepo repository.UserPermissionRepository, resource string, action entity.PermissionAction) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		user, exists := GetUserFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有指定权限
		hasPermission, err := userPermissionRepo.HasPermissionByResourceAction(user.ID, resource, action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permission",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":             "Insufficient permissions",
				"required_resource": resource,
				"required_action":   string(action),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission 要求用户拥有任意一个指定权限的中间件
func RequireAnyPermission(userPermissionRepo repository.UserPermissionRepository, permissionNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		user, exists := GetUserFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有任意一个权限
		hasPermission, err := userPermissionRepo.HasAnyPermission(user.ID, permissionNames)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permission",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":                "Insufficient permissions",
				"required_permissions": permissionNames,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions 要求用户拥有所有指定权限的中间件
func RequireAllPermissions(userPermissionRepo repository.UserPermissionRepository, permissionNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息
		user, exists := GetUserFromContext(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有所有权限
		hasPermission, err := userPermissionRepo.HasAllPermissions(user.ID, permissionNames)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permission",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":                "Insufficient permissions",
				"required_permissions": permissionNames,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
