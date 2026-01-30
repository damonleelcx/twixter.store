package middleware

import (
	"backend/entity"
	"backend/repository"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ViewingCookieName 与前端 twixter_viewing 一致
const ViewingCookieName = "twixter_viewing"
const viewingCookieMaxAge = 365 * 24 * 60 * 60 // 365 days

// ViewingCookieFromRequest 从请求中读取 twixter_viewing cookie（加密值），解密后返回 viewing_light 或 viewing_dark
func ViewingCookieFromRequest(c *gin.Context) string {
	val, _ := c.Cookie(ViewingCookieName)
	return DecryptViewingToken(val)
}

// DecodeViewingParam 解密 query ?viewing= 的 token（加密或旧格式），返回 viewing_light 或 viewing_dark
func DecodeViewingParam(token string) string {
	return DecryptViewingToken(token)
}

// ViewingCookieMiddleware 若 query 带 ?viewing=<token>，将 token 原样写入 cookie（前端/分享链接传的是加密 token）
func ViewingCookieMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("viewing")
		if token != "" && DecryptViewingToken(token) != "" {
			c.SetCookie(ViewingCookieName, token, viewingCookieMaxAge, "/", "", false, false)
		}
		c.Next()
	}
}

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
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireNSFWPermissionForDarkList 检查 feed 列表 category=dark 时的权限
// 当 query category=dark 时：已登录需 can_view_nsfw；未登录则需 cookie twixter_viewing=viewing_dark
func RequireNSFWPermissionForDarkList(userPermissionRepo repository.UserPermissionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.DefaultQuery("category", "light")
		if category != "dark" {
			c.Next()
			return
		}
		user, exists := GetUserFromContext(c)
		if exists {
			hasPermission, err := userPermissionRepo.HasPermission(user.ID, "can_view_nsfw")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permission"})
				c.Abort()
				return
			}
			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
			c.Next()
			return
		}
		// 未登录：允许 cookie twixter_viewing=viewing_dark 或 query ?viewing= 解码为 viewing_dark
		if ViewingCookieFromRequest(c) == "viewing_dark" || DecodeViewingParam(c.Query("viewing")) == "viewing_dark" {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
	}
}

// RequireNSFWPermissionForDarkContent 检查 dark 分类内容的权限中间件
// 如果内容为 dark 分类，则要求用户拥有 can_view_nsfw 权限
func RequireNSFWPermissionForDarkContent(
	contentRepo repository.ContentRepository,
	userPermissionRepo repository.UserPermissionRepository,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路径参数获取内容ID
		idParam := c.Param("id")
		if idParam == "" {
			c.Next() // 如果没有ID参数，继续（可能不是内容路由）
			return
		}

		var contentID uint
		if _, err := fmt.Sscanf(idParam, "%d", &contentID); err != nil {
			c.Next() // ID格式错误，让控制器处理
			return
		}

		// 获取内容
		content, err := contentRepo.GetByID(contentID)
		if err != nil {
			c.Next() // 内容不存在，让控制器处理404
			return
		}

		// 如果内容不是 dark 分类，直接通过
		if !content.IsDark() {
			c.Next()
			return
		}

		// 内容为 dark 分类，需要检查权限
		user, exists := GetUserFromContext(c)
		if exists {
			hasPermission, err := userPermissionRepo.HasPermission(user.ID, "can_view_nsfw")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permission"})
				c.Abort()
				return
			}
			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
			c.Next()
			return
		}
		// 未登录：允许 cookie twixter_viewing=viewing_dark 或 query ?viewing= 解码为 viewing_dark 时访问 dark 内容详情
		if ViewingCookieFromRequest(c) == "viewing_dark" || DecodeViewingParam(c.Query("viewing")) == "viewing_dark" {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
	}
}

// RequireNSFWPermissionForDarkContentByFileID 通过文件ID检查 dark 分类内容的权限中间件
// 如果内容为 dark 分类，则要求用户拥有 can_view_nsfw 权限
// 用于流式传输文件等使用 file_id 的路由
func RequireNSFWPermissionForDarkContentByFileID(
	fileRepo repository.ContentFileRepository,
	contentRepo repository.ContentRepository,
	userPermissionRepo repository.UserPermissionRepository,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路径参数获取文件ID
		fileIDParam := c.Param("file_id")
		if fileIDParam == "" {
			c.Next() // 如果没有file_id参数，继续（可能不是文件路由）
			return
		}

		var fileID uint
		if _, err := fmt.Sscanf(fileIDParam, "%d", &fileID); err != nil {
			c.Next() // ID格式错误，让控制器处理
			return
		}

		// 获取文件记录
		file, err := fileRepo.GetByID(fileID)
		if err != nil {
			c.Next() // 文件不存在，让控制器处理404
			return
		}

		// 获取内容
		content, err := contentRepo.GetByID(file.ContentID)
		if err != nil {
			c.Next() // 内容不存在，让控制器处理404
			return
		}

		// 如果内容不是 dark 分类，直接通过
		if !content.IsDark() {
			c.Next()
			return
		}

		// 内容为 dark 分类，需要检查权限
		user, exists := GetUserFromContext(c)
		if exists {
			hasPermission, err := userPermissionRepo.HasPermission(user.ID, "can_view_nsfw")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permission"})
				c.Abort()
				return
			}
			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
			c.Next()
			return
		}
		// 未登录：允许 cookie 或 query ?viewing= 解码为 viewing_dark 时访问 dark 内容流
		if ViewingCookieFromRequest(c) == "viewing_dark" || DecodeViewingParam(c.Query("viewing")) == "viewing_dark" {
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
	}
}
