package controller

import (
	"backend/entity"
	"backend/middleware"
	"backend/repository"
	"backend/service"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ContentController 内容控制器
type ContentController struct {
	contentService     service.ContentService
	userPermissionRepo repository.UserPermissionRepository
	purchaseRepo       repository.PurchaseRepository
}

// NewContentController 创建内容控制器实例
func NewContentController(contentService service.ContentService, userPermissionRepo repository.UserPermissionRepository, purchaseRepo repository.PurchaseRepository) *ContentController {
	return &ContentController{
		contentService:     contentService,
		userPermissionRepo: userPermissionRepo,
		purchaseRepo:       purchaseRepo,
	}
}

// VideoMetadata 单个视频的元数据
type VideoMetadata struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`                                          // 标签名称列表
	Price       float64  `json:"price"`                                         // 价格（美元，用于单个视频销售，0 表示免费）
	Category    string   `json:"category" binding:"omitempty,oneof=light dark"` // 分类：light（普通）或 dark（NSFW），默认为 light
}

// UploadVideosRequest 批量上传视频请求（用于表单解析）
type UploadVideosRequest struct {
	VideosJSON string `form:"videos" binding:"required"` // JSON 字符串，包含所有视频的元数据
}

// UpdateContentRequest 更新内容请求（PATCH）
type UpdateContentRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Category    string   `json:"category" binding:"required,oneof=light dark"`
	Price       float64  `json:"price" binding:"gte=0"`
	Tags        []string `json:"tags"`
}

// UploadVideos 批量上传视频
// @Summary Upload multiple videos
// @Description Upload up to 10 video files in bulk. Each video requires its own name, description, and tags.
// @Tags content
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param videos formData string true "JSON array of video metadata. Each item should have: name (required), description (optional), tags (optional array of tag names)"
// @Param files formData file true "Video files (max 10, must match the number of video metadata items)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/content/videos/upload [post]
func (cc *ContentController) UploadVideos(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 解析表单
	var req UploadVideosRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// 解析视频元数据 JSON
	var videosMetadata []VideoMetadata
	if err := json.Unmarshal([]byte(req.VideosJSON), &videosMetadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid videos JSON format",
			"details": err.Error(),
		})
		return
	}

	// 获取上传的文件
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to parse multipart form",
			"details": err.Error(),
		})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No files provided",
		})
		return
	}

	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 10 files allowed",
		})
		return
	}

	// 验证视频元数据数量与文件数量匹配
	if len(videosMetadata) != len(files) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Number of video metadata (%d) does not match number of files (%d)", len(videosMetadata), len(files)),
		})
		return
	}

	// 验证文件类型
	for _, file := range files {
		contentType := file.Header.Get("Content-Type")
		if contentType != "video/mp4" && contentType != "video/quicktime" &&
			contentType != "video/x-msvideo" && contentType != "video/webm" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid file type: %s. Only video files are allowed", contentType),
			})
			return
		}
	}

	// 将控制器的 VideoMetadata 转换为服务层的 VideoMetadata
	serviceMetadata := make([]service.VideoMetadata, len(videosMetadata))
	for i, vm := range videosMetadata {
		// 如果未指定分类，默认为 light
		category := vm.Category
		if category == "" {
			category = "light"
		}
		serviceMetadata[i] = service.VideoMetadata{
			Name:        vm.Name,
			Description: vm.Description,
			Tags:        vm.Tags,
			Price:       vm.Price,
			Category:    category,
		}
	}

	// 调用服务上传文件
	contentFiles, err := cc.contentService.UploadVideos(userBase.ID, files, serviceMetadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to upload videos",
			"details": err.Error(),
		})
		return
	}

	// 使内容相关缓存失效（新上传的内容）
	if middleware.GlobalCacheMiddleware != nil {
		for _, file := range contentFiles {
			middleware.GlobalCacheMiddleware.InvalidateContentCache(file.ContentID)
		}
	}

	// 构建响应
	fileResponses := make([]gin.H, 0, len(contentFiles))
	for _, file := range contentFiles {
		fileResponses = append(fileResponses, gin.H{
			"id":                file.ID,
			"file_name":         file.FileName,
			"file_index":        file.FileIndex,
			"original_file_url": file.OriginalFileURL,
			"stage":             file.Stage,
			"created_at":        file.CreatedAt,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Videos uploaded successfully",
		"files":   fileResponses,
	})
}

// GetContent 获取内容详情
// @Summary Get content details
// @Description Get content by ID. Dark category content requires can_view_nsfw permission.
// @Tags content
// @Security BearerAuth
// @Produce json
// @Param id path int true "Content ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// contentIDUri 用于绑定路径参数 :id
type contentIDUri struct {
	ID uint `uri:"id" binding:"required"`
}

// @Router /api/content/{id} [get]
func (cc *ContentController) GetContent(c *gin.Context) {
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid content ID",
		})
		return
	}
	id := params.ID

	content, err := cc.contentService.GetContent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Content not found",
		})
		return
	}

	// 权限检查已由中间件 RequireNSFWPermissionForDarkContent 处理

	// 获取所有文件
	files, err := cc.contentService.GetContentFiles(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get content files",
		})
		return
	}

	fileResponses := make([]gin.H, 0, len(files))
	for _, file := range files {
		fileResponses = append(fileResponses, gin.H{
			"id":                  file.ID,
			"file_name":           file.FileName,
			"file_index":          file.FileIndex,
			"original_file_url":   file.OriginalFileURL,
			"gif_file_url":        file.GifFileURL,
			"transcoded_file_url": file.TranscodedFileURL,
			"stage":               file.Stage,
			"width":               file.Width,
			"height":              file.Height,
			"duration":            file.Duration,
		})
	}

	purchased := false
	bookmarked := false
	if user, exists := middleware.GetUserFromContext(c); exists {
		purchased, _ = cc.contentService.UserCanViewContent(user.ID, content.ID)
		bookmarked, _ = cc.contentService.IsBookmarked(user.ID, content.ID)
	}

	authorUsername, authorAvatar := cc.contentService.GetAuthorForUserID(content.UploadedBy)
	if authorUsername == "" {
		authorUsername = content.Name
	}

	tagNames, _ := cc.contentService.GetContentTagNames(id)

	c.JSON(http.StatusOK, gin.H{
		"content": gin.H{
			"id":              content.ID,
			"name":            content.Name,
			"description":     content.Description,
			"type":            content.Type,
			"status":          content.Status,
			"category":        content.Category,
			"price":           content.Price,
			"created_at":      content.CreatedAt,
			"purchased":       purchased,
			"bookmarked":      bookmarked,
			"author_username": authorUsername,
			"author_avatar":   authorAvatar,
			"tags":            tagNames,
		},
		"files": fileResponses,
	})
}

// GetViewingToken 返回用于 URL/cookie 的加密 viewing token（mode=dark|light），前端不持有密钥
func (cc *ContentController) GetViewingToken(c *gin.Context) {
	mode := c.DefaultQuery("mode", "light")
	var plain string
	if mode == "dark" {
		plain = "viewing_dark"
	} else {
		plain = "viewing_light"
	}
	token := middleware.EncryptViewingToken(plain)
	if token == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Viewing token not available",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// UpdateContent 更新内容元数据（需 can_edit_content 权限）
func (cc *ContentController) UpdateContent(c *gin.Context) {
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content ID"})
		return
	}
	var req UpdateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}
	if err := cc.contentService.UpdateContent(params.ID, req.Name, req.Description, req.Category, req.Price, req.Tags); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Content not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update content", "details": err.Error()})
		return
	}
	if middleware.GlobalCacheMiddleware != nil {
		middleware.GlobalCacheMiddleware.InvalidateContentCache(params.ID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Content updated successfully"})
}

// DeleteContent 删除内容（需 can_delete_content 权限，仅 admin）
func (cc *ContentController) DeleteContent(c *gin.Context) {
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content ID"})
		return
	}
	if err := cc.contentService.DeleteContent(params.ID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Content not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete content", "details": err.Error()})
		return
	}
	if middleware.GlobalCacheMiddleware != nil {
		middleware.GlobalCacheMiddleware.InvalidateContentCache(params.ID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Content deleted successfully"})
}

// ListContent 分页列出 feed 内容。支持按分类、按标签(tag)、或模糊搜索(q)。dark 分类由 RequireNSFWPermissionForDarkList 中间件校验。
// ListFeedByTag 仅允许拥有 can_search_tags 权限的用户使用；搜索时无 can_view_nsfw 的用户只能搜 light 分类。
func (cc *ContentController) ListContent(c *gin.Context) {
	category := c.DefaultQuery("category", "light")
	if category != "light" && category != "dark" {
		category = ""
	}
	tag := strings.TrimSpace(c.Query("tag"))
	q := strings.TrimSpace(c.Query("q"))
	cursor, _ := strconv.Atoi(c.DefaultQuery("cursor", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := cursor * limit
	userID := uint(0)
	var user *entity.UserBase
	if u, exists := middleware.GetUserFromContext(c); exists {
		user = u
		userID = u.ID
	}

	// ListFeedByTag: only users with can_search_tags may use tag filter
	if tag != "" {
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in required to search by tag"})
			return
		}
		hasSearchTags, errPerm := cc.userPermissionRepo.HasPermission(user.ID, "can_search_tags")
		if errPerm != nil || !hasSearchTags {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission required to search by tag (can_search_tags)"})
			return
		}
	}

	var items []service.ListFeedItem
	var err error
	if tag != "" {
		items, err = cc.contentService.ListFeedByTag(tag, userID, limit, offset)
	} else if q != "" {
		// Search: users without can_view_nsfw get light only; users with can_view_nsfw search all categories
		searchCategory := ""
		if user == nil {
			searchCategory = "light"
		} else {
			hasNSFW, _ := cc.userPermissionRepo.HasPermission(user.ID, "can_view_nsfw")
			if !hasNSFW {
				searchCategory = "light"
			}
			// hasNSFW: searchCategory stays "" so they search all content (light + dark)
		}
		items, err = cc.contentService.ListFeedSearch(q, searchCategory, userID, limit, offset)
	} else {
		if category == "" {
			category = "light"
		}
		items, err = cc.contentService.ListFeed(category, userID, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list content",
			"details": err.Error(),
		})
		return
	}
	nextCursor := cursor + 1
	hasMore := len(items) == limit
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// ListLibrary 获取当前用户已购买的内容列表（Library 页）
func (cc *ContentController) ListLibrary(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	cursor, _ := strconv.Atoi(c.DefaultQuery("cursor", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := cursor * limit
	items, err := cc.contentService.ListPurchasedContent(user.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list library",
			"details": err.Error(),
		})
		return
	}
	nextCursor := cursor + 1
	hasMore := len(items) == limit
	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// AddBookmark 添加书签
func (cc *ContentController) AddBookmark(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content ID"})
		return
	}
	if err := cc.contentService.AddBookmark(user.ID, params.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bookmarked"})
}

// RemoveBookmark 移除书签
func (cc *ContentController) RemoveBookmark(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content ID"})
		return
	}
	if err := cc.contentService.RemoveBookmark(user.ID, params.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Removed"})
}

// ListBookmarks 获取当前用户书签的内容列表（Bookmarks 页）
func (cc *ContentController) ListBookmarks(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	cursor, _ := strconv.Atoi(c.DefaultQuery("cursor", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := cursor * limit
	items, err := cc.contentService.ListBookmarkedContent(user.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list bookmarks",
			"details": err.Error(),
		})
		return
	}
	nextCursor := cursor + 1
	hasMore := len(items) == limit
	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// StreamGifPreview 流式返回 GIF 预览：已购买返回原图，未购买返回后端模糊后的 JPEG
func (cc *ContentController) StreamGifPreview(c *gin.Context) {
	fileIDStr := c.Param("file_id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file ID",
		})
		return
	}
	userID := uint(0)
	if user, exists := middleware.GetUserFromContext(c); exists {
		userID = user.ID
	}
	reader, contentType, err := cc.contentService.GetGifPreview(uint(fileID), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Preview not found",
		})
		return
	}
	defer reader.Close()
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "private, max-age=300")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

// StreamTranscodedFile 流式传输转码文件（HLS .m3u8 播放列表）
// @Summary Stream transcoded HLS playlist file
// @Description Stream HLS playlist (.m3u8) file by file ID. Supports HLS streaming with .m3u8 and .ts segment files.
// @Tags content
// @Produce application/vnd.apple.mpegurl
// @Param file_id path int true "File ID"
// @Param segment query string false "Segment file name (for .ts files)"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/content/files/{file_id}/stream [get]
func (cc *ContentController) StreamTranscodedFile(c *gin.Context) {
	fileIDStr := c.Param("file_id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file ID",
		})
		return
	}

	// 获取转码文件信息
	file, err := cc.contentService.GetTranscodedFile(uint(fileID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Transcoded file not found",
		})
		return
	}

	// 已购买或有效会员可观看视频流（light 与 dark 内容均适用，UserCanViewContent 不区分 category）
	user, hasUser := middleware.GetUserFromContext(c)
	if !hasUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	canView, err := cc.contentService.UserCanViewContent(user.ID, file.ContentID)
	if err != nil || !canView {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// 检查是否请求的是分片文件（.ts）
	segment := c.Query("segment")
	if segment != "" {
		// 请求的是 .ts 分片文件
		// 构建分片文件的 S3 路径（假设分片文件与 .m3u8 在同一目录）
		segmentPath := cc.contentService.GetSegmentFilePath(file.TranscodedFilePath, segment)

		reader, _, err := cc.contentService.StreamFileFromS3(segmentPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Segment file not found",
			})
			return
		}
		defer reader.Close()

		// 设置响应头（.ts 文件的 Content-Type）
		c.Header("Content-Type", "video/mp2t")
		c.Header("Accept-Ranges", "bytes")
		c.Header("Cache-Control", "public, max-age=3600")

		// 流式传输分片文件
		c.DataFromReader(http.StatusOK, -1, "video/mp2t", reader, nil)
		return
	}

	// 请求的是 .m3u8 播放列表文件：记录观看（每次开始拉流时记一次，不记 .ts 分片请求）
	if err := cc.contentService.RecordView(file.ContentID, user.ID); err == nil && middleware.GlobalCacheMiddleware != nil {
		middleware.GlobalCacheMiddleware.InvalidateContentCache(file.ContentID)
	}

	// 从S3流式传输 .m3u8 文件
	reader, _, err := cc.contentService.StreamFileFromS3(file.TranscodedFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to stream playlist file",
		})
		return
	}
	defer reader.Close()

	// 设置响应头（.m3u8 文件的 Content-Type）
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")

	// 流式传输 .m3u8 文件
	c.DataFromReader(http.StatusOK, -1, "application/vnd.apple.mpegurl", reader, nil)
}

// WatchProgressRequest 观看进度请求体
type WatchProgressRequest struct {
	WatchTimeSeconds float64 `json:"watch_time_seconds" binding:"required,gte=0"` // 本次观看时长（秒）
	DurationSeconds  float64 `json:"duration_seconds"`                            // 视频总时长（秒），可选，用于计算完成率
}

// RecordWatchProgress 记录观看进度（视频流播放时由前端上报，用于更新 TotalWatchTime / AverageWatchTime / CompletionRate）
// @Summary Record watch progress
// @Description Record watch time and update completion rate for analytics
// @Tags content
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Content ID"
// @Param body body WatchProgressRequest true "Watch progress"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/content/{id}/watch-progress [post]
func (cc *ContentController) RecordWatchProgress(c *gin.Context) {
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content ID"})
		return
	}
	var req WatchProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userBase := user.(*entity.UserBase)
	duration := req.DurationSeconds
	if duration <= 0 {
		// 从内容首个文件的 duration 获取
		files, err := cc.contentService.GetContentFiles(params.ID)
		if err == nil && len(files) > 0 && files[0].Duration != nil && *files[0].Duration > 0 {
			duration = *files[0].Duration
		}
	}
	if err := cc.contentService.RecordWatchProgress(params.ID, userBase.ID, req.WatchTimeSeconds, duration); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if middleware.GlobalCacheMiddleware != nil {
		middleware.GlobalCacheMiddleware.InvalidateContentCache(params.ID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Watch progress recorded"})
}

// RecordContentView 记录内容观看
// @Summary Record content view
// @Description Record a view/play event for content
// @Tags content
// @Security BearerAuth
// @Produce json
// @Param id path int true "Content ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/content/{id}/view [post]
func (cc *ContentController) RecordContentView(c *gin.Context) {
	// 获取内容ID
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid content ID",
		})
		return
	}
	id := params.ID

	// 未登录（仅凭 viewing cookie 访问）时不记录观看，直接返回成功
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"message": "View recorded successfully",
		})
		return
	}

	userBase := user.(*entity.UserBase)

	// 记录观看
	if err := cc.contentService.RecordView(id, userBase.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to record view",
			"details": err.Error(),
		})
		return
	}

	// 使内容分析数据缓存失效
	if middleware.GlobalCacheMiddleware != nil {
		middleware.GlobalCacheMiddleware.InvalidateContentCache(id)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "View recorded successfully",
	})
}

// GetContentAnalytics 获取内容分析数据
// @Summary Get content analytics
// @Description Get analytics data for content
// @Tags content
// @Security BearerAuth
// @Produce json
// @Param id path int true "Content ID"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/content/{id}/analytics [get]
func (cc *ContentController) GetContentAnalytics(c *gin.Context) {
	// 获取内容ID
	var params contentIDUri
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid content ID",
		})
		return
	}
	id := params.ID

	// 获取日期范围参数
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	// 如果没有提供日期，默认使用最近30天
	if startDate == "" || endDate == "" {
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	// 获取分析数据
	analytics, err := cc.contentService.GetContentAnalytics(id, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get analytics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content_id": id,
		"start_date": startDate,
		"end_date":   endDate,
		"analytics":  analytics,
	})
}
