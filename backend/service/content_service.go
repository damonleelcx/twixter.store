package service

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// VideoMetadata 视频元数据
type VideoMetadata struct {
	Name        string
	Description string
	Tags        []string // 标签名称列表
	Price       float64  // 价格（美元，用于单个视频销售，0 表示免费）
	Category    string   // 分类：light（普通）或 dark（NSFW），默认为 light
}

// ContentService 内容服务接口
type ContentService interface {
	// UploadVideos 批量上传视频（最多10个），每个视频有独立的元数据
	UploadVideos(userID uint, files []*multipart.FileHeader, videosMetadata []VideoMetadata) ([]*entity.ContentFile, error)

	// GetContent 获取内容
	GetContent(contentID uint) (*entity.Content, error)
	// GetContentTagNames 获取内容的标签名称列表
	GetContentTagNames(contentID uint) ([]string, error)

	// GetContentFiles 获取内容的所有文件
	GetContentFiles(contentID uint) ([]*entity.ContentFile, error)

	// UpdateFileStage 更新文件处理阶段
	UpdateFileStage(fileID uint, stage entity.FileProcessingStage) error

	// GetTranscodedFile 获取转码文件信息
	GetTranscodedFile(fileID uint) (*entity.ContentFile, error)

	// StreamFileFromS3 从S3流式传输文件
	StreamFileFromS3(s3Key string) (io.ReadCloser, string, error)

	// GetSegmentFilePath 获取分片文件的S3路径
	GetSegmentFilePath(m3u8Path string, segmentName string) string

	// RecordView 记录内容观看
	RecordView(contentID uint, userID uint) error
	// RecordWatchProgress 记录观看进度（增加 TotalWatchTime、AverageWatchTime、CompletionRate）
	RecordWatchProgress(contentID uint, userID uint, watchTimeSeconds float64, contentDurationSeconds float64) error

	// GetContentAnalytics 获取内容分析数据
	GetContentAnalytics(contentID uint, startDate, endDate string) (map[string]interface{}, error)

	// ListFeed 按分类分页列出 feed 内容（仅 ready），含首文件 gif 与当前用户是否已购买。sortBy: "view_count" 或 "created_at"（默认）。freeOnly 为 true 时仅返回 price=0 的免费内容
	ListFeed(category string, userID uint, limit, offset int, sortBy string, freeOnly bool) ([]ListFeedItem, error)
	// ListFeedByTag 按标签名列出 feed 内容（仅 ready）；freeOnly 为 true 时仅返回 price=0 的免费内容
	ListFeedByTag(tagName string, userID uint, limit, offset int, sortBy string, freeOnly bool) ([]ListFeedItem, error)
	// ListFeedSearch 模糊搜索 name/description 列出 feed（仅 ready），category 为空时搜全部；freeOnly 为 true 时仅返回 price=0 的免费内容
	ListFeedSearch(q string, category string, userID uint, limit, offset int, sortBy string, freeOnly bool) ([]ListFeedItem, error)

	// GetGifPreview 返回 GIF 预览：可观看（已购买或有效会员）则流式返回正常 GIF，否则返回模糊 GIF
	GetGifPreview(fileID uint, userID uint) (body io.ReadCloser, contentType string, err error)

	// GetAuthorForUserID 根据上传者 ID 返回作者用户名和头像 URL（用于帖子详情等）
	GetAuthorForUserID(userID uint) (username string, avatar string)

	// ListPurchasedContent 返回当前用户已购买的内容列表（Library 页），格式与 feed 一致
	ListPurchasedContent(userID uint, limit, offset int) ([]ListFeedItem, error)

	// AddBookmark 添加书签
	AddBookmark(userID, contentID uint) error
	// RemoveBookmark 移除书签
	RemoveBookmark(userID, contentID uint) error
	// IsBookmarked 当前用户是否已书签该内容
	IsBookmarked(userID, contentID uint) (bool, error)
	// ListBookmarkedContent 返回当前用户书签的内容列表（Bookmarks 页）
	ListBookmarkedContent(userID uint, limit, offset int) ([]ListFeedItem, error)

	// UserCanViewContent 用户是否可观看该内容（已购买或有效会员）
	UserCanViewContent(userID uint, contentID uint) (bool, error)

	// UpdateContent 更新内容元数据（需 can_edit_content 权限）
	UpdateContent(contentID uint, name, description, category string, price float64, tagNames []string) error
	// DeleteContent 删除内容（需 can_delete_content 权限，仅 admin）
	DeleteContent(contentID uint) error
}

// ListFeedItem feed 列表单项（含作者、时间戳、标签，Twitter 风格）
// PreviewGifURL 按权限返回：已购买为正常 GIF URL，未购买为模糊 GIF URL（上传时生成）
type ListFeedItem struct {
	ID               uint     `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Category         string   `json:"category"`
	Price            float64  `json:"price"`
	PreviewGifURL    string   `json:"preview_gif_url"`
	PreviewGifBase64 string   `json:"preview_gif_base64,omitempty"` // 保留兼容，不再使用
	FirstFileID      uint     `json:"first_file_id"`
	PreviewWidth     int      `json:"preview_width,omitempty"`  // 首文件宽，用于前端正确宽高比
	PreviewHeight    int      `json:"preview_height,omitempty"` // 首文件高
	Duration         *float64 `json:"duration,omitempty"`       // 时长（秒），用于前端显示
	Purchased        bool     `json:"purchased"`
	Bookmarked       bool     `json:"bookmarked,omitempty"`
	CreatedAt        string   `json:"created_at"`
	AuthorUsername   string   `json:"author_username"`
	AuthorAvatar     string   `json:"author_avatar,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

// contentService 内容服务实现
type contentService struct {
	contentRepo         repository.ContentRepository
	fileRepo            repository.ContentFileRepository
	tagRepo             repository.TagRepository
	contentTagRepo      repository.ContentTagRepository
	contentBookmarkRepo repository.ContentBookmarkRepository
	analyticsRepo       repository.AnalyticsRepository
	purchaseRepo        repository.PurchaseRepository
	userRepo            repository.UserRepository
	userPermissionRepo  repository.UserPermissionRepository
	s3Service           S3Service
	kafkaService        KafkaService
}

// NewContentService 创建内容服务实例
func NewContentService(
	contentRepo repository.ContentRepository,
	fileRepo repository.ContentFileRepository,
	tagRepo repository.TagRepository,
	contentTagRepo repository.ContentTagRepository,
	contentBookmarkRepo repository.ContentBookmarkRepository,
	analyticsRepo repository.AnalyticsRepository,
	purchaseRepo repository.PurchaseRepository,
	userRepo repository.UserRepository,
	userPermissionRepo repository.UserPermissionRepository,
	s3Service S3Service,
	kafkaService KafkaService,
) ContentService {
	return &contentService{
		contentRepo:         contentRepo,
		fileRepo:            fileRepo,
		tagRepo:             tagRepo,
		contentTagRepo:      contentTagRepo,
		contentBookmarkRepo: contentBookmarkRepo,
		analyticsRepo:       analyticsRepo,
		purchaseRepo:        purchaseRepo,
		userRepo:            userRepo,
		userPermissionRepo:  userPermissionRepo,
		s3Service:           s3Service,
		kafkaService:        kafkaService,
	}
}

// ensureTagExists 确保标签存在，如果不存在则创建，返回标签ID
func (s *contentService) ensureTagExists(tagName string) (uint, error) {
	// 清理标签名称（去除首尾空格）
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return 0, errors.New("tag name cannot be empty")
	}

	// 尝试查找现有标签
	tag, err := s.tagRepo.GetByName(tagName)
	if err == nil {
		// 标签已存在，返回其ID
		return tag.ID, nil
	}
	// 如果错误不是记录未找到，说明查询出错
	if err != gorm.ErrRecordNotFound {
		// 查询出错
		return 0, fmt.Errorf("failed to query tag: %w", err)
	}

	// 标签不存在，创建新标签
	// 生成 slug（简单的 slug 生成：转小写，空格替换为连字符）
	slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))

	// 确保 slug 唯一
	originalSlug := slug
	counter := 1
	for {
		existingTag, err := s.tagRepo.GetBySlug(slug)
		if err == gorm.ErrRecordNotFound {
			break // slug 可用
		}
		if err != nil {
			return 0, fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if existingTag != nil {
			// slug 已存在，尝试添加数字后缀
			slug = fmt.Sprintf("%s-%d", originalSlug, counter)
			counter++
		} else {
			break
		}
	}

	newTag := &entity.Tag{
		Name: tagName,
		Slug: slug,
	}

	if err := s.tagRepo.Create(newTag); err != nil {
		return 0, fmt.Errorf("failed to create tag: %w", err)
	}

	return newTag.ID, nil
}

// createdUpload 单次上传中已创建的资源，用于失败时回滚
type createdUpload struct {
	contentID uint
	fileID    uint
	s3Key     string
}

// UploadVideos 批量上传视频（最多10个），每个视频有独立的元数据。任一步失败会回滚本请求内已创建的 S3 与 DB 记录。
func (s *contentService) UploadVideos(userID uint, files []*multipart.FileHeader, videosMetadata []VideoMetadata) ([]*entity.ContentFile, error) {
	if len(files) == 0 {
		return nil, errors.New("no files provided")
	}
	if len(files) > 10 {
		return nil, errors.New("maximum 10 files allowed")
	}
	if len(videosMetadata) != len(files) {
		return nil, fmt.Errorf("number of video metadata (%d) does not match number of files (%d)", len(videosMetadata), len(files))
	}

	contentFiles := make([]*entity.ContentFile, 0, len(files))
	created := make([]createdUpload, 0, len(files))

	for i, fileHeader := range files {
		metadata := videosMetadata[i]

		if strings.TrimSpace(metadata.Name) == "" {
			s.rollbackUploads(created)
			return nil, fmt.Errorf("video %d: name is required", i+1)
		}
		category := entity.ContentCategoryLight
		if metadata.Category == "dark" {
			category = entity.ContentCategoryDark
		}
		content := &entity.Content{
			Name:        strings.TrimSpace(metadata.Name),
			Description: strings.TrimSpace(metadata.Description),
			Type:        entity.ContentTypeVideo,
			Status:      entity.ContentStatusPending,
			UploadedBy:  userID,
			Price:       metadata.Price,
			Category:    category,
		}
		if err := s.contentRepo.Create(content); err != nil {
			s.rollbackUploads(created)
			return nil, fmt.Errorf("failed to create content for video %d: %w", i+1, err)
		}
		created = append(created, createdUpload{contentID: content.ID, fileID: 0, s3Key: ""})

		if len(metadata.Tags) > 0 {
			tagIDs := make([]uint, 0, len(metadata.Tags))
			for _, tagName := range metadata.Tags {
				tagID, err := s.ensureTagExists(tagName)
				if err != nil {
					s.rollbackUploads(created)
					return nil, fmt.Errorf("failed to process tag '%s' for video %d: %w", tagName, i+1, err)
				}
				if tagID > 0 {
					tagIDs = append(tagIDs, tagID)
				}
			}
			if len(tagIDs) > 0 {
				if err := s.contentTagRepo.AddTagsToContent(content.ID, tagIDs); err != nil {
					s.rollbackUploads(created)
					return nil, fmt.Errorf("failed to add tags to content %d: %w", content.ID, err)
				}
			}
		}

		file, err := fileHeader.Open()
		if err != nil {
			s.rollbackUploads(created)
			return nil, fmt.Errorf("failed to open file %s: %w", fileHeader.Filename, err)
		}
		timestamp := time.Now().Unix()
		s3Key := fmt.Sprintf("videos/%d/%d_%s", userID, timestamp, filepath.Base(fileHeader.Filename))
		fileURL, err := s.s3Service.UploadFile("", s3Key, file, fileHeader.Header.Get("Content-Type"))
		file.Close()
		if err != nil {
			s.rollbackUploads(created)
			return nil, fmt.Errorf("failed to upload file to S3: %w", err)
		}
		created[len(created)-1].s3Key = s3Key

		fileSize := fileHeader.Size
		contentFile := &entity.ContentFile{
			ContentID:        content.ID,
			FileName:         fileHeader.Filename,
			FileIndex:        0,
			OriginalFilePath: s3Key,
			OriginalFileURL:  fileURL,
			OriginalFileSize: fileSize,
			MimeType:         fileHeader.Header.Get("Content-Type"),
			Stage:            entity.StageUploaded,
		}
		if err := s.fileRepo.Create(contentFile); err != nil {
			s.rollbackUploads(created)
			return nil, fmt.Errorf("failed to create file record: %w", err)
		}
		created[len(created)-1].fileID = contentFile.ID
		contentFiles = append(contentFiles, contentFile)

		content.Status = entity.ContentStatusProcessing
		if err := s.contentRepo.Update(content); err != nil {
			s.rollbackUploads(created)
			return nil, fmt.Errorf("failed to update content status: %w", err)
		}

		message := &KafkaMessage{
			Type:      "gif_generation",
			ContentID: content.ID,
			FileID:    contentFile.ID,
			Stage:     string(entity.StageUploaded),
			Data: map[string]interface{}{
				"s3_key":   s3Key,
				"file_url": fileURL,
			},
		}
		if err := s.kafkaService.SendMessage(TopicGifGeneration, message); err != nil {
			fmt.Printf("Failed to send Kafka message for file %d: %v\n", contentFile.ID, err)
		}
	}
	return contentFiles, nil
}

// rollbackUploads 回滚本请求内已创建的资源：删除 S3 对象、文件记录、内容标签、内容记录（逆序）
func (s *contentService) rollbackUploads(created []createdUpload) {
	for i := len(created) - 1; i >= 0; i-- {
		c := created[i]
		if c.s3Key != "" {
			_ = s.s3Service.DeleteFile("", c.s3Key)
		}
		if c.fileID != 0 {
			_ = s.fileRepo.Delete(c.fileID)
		}
		if c.contentID != 0 {
			_ = s.contentTagRepo.RemoveAllTagsFromContent(c.contentID)
			_ = s.contentRepo.Delete(c.contentID)
		}
	}
}

// GetContent 获取内容
func (s *contentService) GetContent(contentID uint) (*entity.Content, error) {
	return s.contentRepo.GetByID(contentID)
}

// GetContentTagNames 获取内容的标签名称列表
func (s *contentService) GetContentTagNames(contentID uint) ([]string, error) {
	tags, err := s.contentTagRepo.GetTagsByContentID(contentID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names, nil
}

// UpdateContent 更新内容元数据（name, description, category, price, tags）；tagNames 为 nil 时不改标签
func (s *contentService) UpdateContent(contentID uint, name, description, category string, price float64, tagNames []string) error {
	content, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}
	content.Name = strings.TrimSpace(name)
	content.Description = strings.TrimSpace(description)
	if category == "dark" || category == "light" {
		content.Category = entity.ContentCategory(category)
	}
	if price >= 0 {
		content.Price = price
	}
	if err := s.contentRepo.Update(content); err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}
	if tagNames != nil {
		tagIDs := make([]uint, 0, len(tagNames))
		for _, tagName := range tagNames {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			tagID, err := s.ensureTagExists(tagName)
			if err != nil {
				return fmt.Errorf("failed to process tag %q: %w", tagName, err)
			}
			if tagID > 0 {
				tagIDs = append(tagIDs, tagID)
			}
		}
		if err := s.contentTagRepo.ReplaceContentTags(contentID, tagIDs); err != nil {
			return fmt.Errorf("failed to update content tags: %w", err)
		}
	}
	return nil
}

// DeleteContent 删除内容（从 S3 删除文件后软删除 DB 记录；需 can_delete_content 权限）
func (s *contentService) DeleteContent(contentID uint) error {
	if _, err := s.contentRepo.GetByID(contentID); err != nil {
		return fmt.Errorf("content not found: %w", err)
	}
	files, err := s.fileRepo.GetByContentID(contentID)
	if err != nil {
		return fmt.Errorf("failed to get content files: %w", err)
	}
	// 从 S3 删除每个文件的原始、GIF、模糊 GIF、转码（m3u8 + segments）
	for _, f := range files {
		if f.OriginalFilePath != "" {
			_ = s.s3Service.DeleteFile("", f.OriginalFilePath)
		}
		if f.GifFilePath != "" {
			_ = s.s3Service.DeleteFile("", f.GifFilePath)
		}
		if f.GifBlurFilePath != "" {
			_ = s.s3Service.DeleteFile("", f.GifBlurFilePath)
		}
		if f.TranscodedFilePath != "" {
			prefix := strings.TrimSuffix(f.TranscodedFilePath, ".m3u8")
			keys, listErr := s.s3Service.ListKeysByPrefix("", prefix)
			if listErr == nil {
				for _, key := range keys {
					_ = s.s3Service.DeleteFile("", key)
				}
			} else {
				_ = s.s3Service.DeleteFile("", f.TranscodedFilePath)
			}
		}
	}
	if err := s.contentTagRepo.RemoveAllTagsFromContent(contentID); err != nil {
		return fmt.Errorf("failed to remove content tags: %w", err)
	}
	for _, f := range files {
		_ = s.fileRepo.Delete(f.ID)
	}
	if err := s.contentRepo.Delete(contentID); err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}
	return nil
}

// GetContentFiles 获取内容的所有文件
func (s *contentService) GetContentFiles(contentID uint) ([]*entity.ContentFile, error) {
	return s.fileRepo.GetByContentID(contentID)
}

// UpdateFileStage 更新文件处理阶段
func (s *contentService) UpdateFileStage(fileID uint, stage entity.FileProcessingStage) error {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	file.Stage = stage
	if stage == entity.StageCompleted || stage == entity.StageFailed {
		now := time.Now()
		file.ProcessedAt = &now
	}

	return s.fileRepo.Update(file)
}

// GetTranscodedFile 获取转码文件信息
func (s *contentService) GetTranscodedFile(fileID uint) (*entity.ContentFile, error) {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	// 检查转码文件是否存在
	if file.TranscodedFilePath == "" {
		return nil, fmt.Errorf("transcoded file not found for file ID %d", fileID)
	}

	return file, nil
}

// StreamFileFromS3 从S3流式传输文件
func (s *contentService) StreamFileFromS3(s3Key string) (io.ReadCloser, string, error) {
	return s.s3Service.StreamFile("", s3Key)
}

// GetSegmentFilePath 获取分片文件的S3路径
// .ts 分片文件与 .m3u8 文件在同一目录下。使用 path 包保证 S3 key 始终为正斜杠（Windows 下 filepath 会产出反斜杠导致 S3 找不到文件）。
func (s *contentService) GetSegmentFilePath(m3u8Path string, segmentName string) string {
	dir := path.Dir(m3u8Path)
	actualSegmentName := segmentName
	if idx := strings.Index(segmentName, "?"); idx != -1 {
		actualSegmentName = segmentName[:idx]
	}
	return path.Join(dir, actualSegmentName)
}

// RecordView 记录内容观看
func (s *contentService) RecordView(contentID uint, userID uint) error {
	// 检查内容是否存在
	_, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}

	// 记录观看次数
	if err := s.analyticsRepo.IncrementViews(contentID); err != nil {
		return fmt.Errorf("failed to increment views: %w", err)
	}

	// 记录播放次数（视频内容）
	if err := s.analyticsRepo.IncrementPlayCount(contentID); err != nil {
		return fmt.Errorf("failed to increment play count: %w", err)
	}

	return nil
}

// RecordWatchProgress 记录观看进度（在视频流播放时由前端上报）
func (s *contentService) RecordWatchProgress(contentID uint, userID uint, watchTimeSeconds float64, contentDurationSeconds float64) error {
	if watchTimeSeconds <= 0 {
		return nil
	}
	// 检查内容存在且用户可观看（可选，避免未购买用户刷数据）
	_, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}
	canView, err := s.UserCanViewContent(userID, contentID)
	if err != nil {
		return fmt.Errorf("user cannot view content: %w", err)
	}
	if !canView {
		return errors.New("insufficient permissions to view content")
	}
	if err := s.analyticsRepo.AddWatchTime(contentID, watchTimeSeconds); err != nil {
		return fmt.Errorf("failed to add watch time: %w", err)
	}
	if contentDurationSeconds > 0 {
		if err := s.analyticsRepo.UpdateCompletionRate(contentID, contentDurationSeconds); err != nil {
			return fmt.Errorf("failed to update completion rate: %w", err)
		}
	}
	return nil
}

// GetContentAnalytics 获取内容分析数据
func (s *contentService) GetContentAnalytics(contentID uint, startDate, endDate string) (map[string]interface{}, error) {
	// 检查内容是否存在
	_, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return nil, fmt.Errorf("content not found: %w", err)
	}

	// 获取日期范围内的分析数据
	analyticsList, err := s.analyticsRepo.GetByContentAndDateRange(contentID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	// 计算聚合统计
	totalViews, err := s.analyticsRepo.GetTotalViews(contentID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get total views: %w", err)
	}

	totalLikes, err := s.analyticsRepo.GetTotalLikes(contentID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get total likes: %w", err)
	}

	avgWatchTime, err := s.analyticsRepo.GetAverageWatchTime(contentID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get average watch time: %w", err)
	}

	// 构建响应数据
	result := map[string]interface{}{
		"total_views":        totalViews,
		"total_likes":        totalLikes,
		"average_watch_time": avgWatchTime,
		"daily_analytics":    make([]map[string]interface{}, 0, len(analyticsList)),
	}

	// 添加每日分析数据
	for _, analytics := range analyticsList {
		dailyData := map[string]interface{}{
			"date":               analytics.Date,
			"views":              analytics.Views,
			"unique_views":       analytics.UniqueViews,
			"likes":              analytics.Likes,
			"dislikes":           analytics.Dislikes,
			"shares":             analytics.Shares,
			"comments":           analytics.Comments,
			"downloads":          analytics.Downloads,
			"play_count":         analytics.PlayCount,
			"average_watch_time": analytics.AverageWatchTime,
			"total_watch_time":   analytics.TotalWatchTime,
			"completion_rate":    analytics.CompletionRate,
			"engagement_rate":    analytics.EngagementRate,
		}
		result["daily_analytics"] = append(result["daily_analytics"].([]map[string]interface{}), dailyData)
	}

	return result, nil
}

// randomUsername 为每条帖子生成一个随机显示用户名
func randomUsername() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "user_" + string(b)
}

// buildListFeedItems 将 contents 转为 ListFeedItem 列表（共用逻辑）；按权限返回正常或模糊 GIF URL
func (s *contentService) buildListFeedItems(contents []entity.Content, userID uint) ([]ListFeedItem, error) {
	out := make([]ListFeedItem, 0, len(contents))
	for _, c := range contents {
		authorUsername := randomUsername()
		files, err := s.fileRepo.GetByContentID(c.ID)
		if err != nil || len(files) == 0 {
			purchased := false
			bookmarked := false
			if userID > 0 {
				purchased, _ = s.UserCanViewContent(userID, c.ID)
				if s.contentBookmarkRepo != nil {
					bookmarked, _ = s.contentBookmarkRepo.Exists(userID, c.ID)
				}
			}
			tagNames, _ := s.GetContentTagNames(c.ID)
			dur := c.Duration
			out = append(out, ListFeedItem{
				ID:             c.ID,
				Name:           c.Name,
				Description:    c.Description,
				Category:       string(c.Category),
				Price:          c.Price,
				Duration:       dur,
				CreatedAt:      c.CreatedAt.Format(time.RFC3339),
				AuthorUsername: authorUsername,
				Purchased:      purchased,
				Bookmarked:     bookmarked,
				Tags:           tagNames,
			})
			continue
		}
		first := files[0]
		purchased := false
		bookmarked := false
		if userID > 0 {
			purchased, _ = s.UserCanViewContent(userID, c.ID)
			bookmarked, _ = s.contentBookmarkRepo.Exists(userID, c.ID)
		}
		// 已购买用正常 GIF URL，未购买用上传时生成的模糊 GIF URL；两者都空时用内容缩略图兜底
		previewGif := first.GifFileURL
		if !purchased && first.GifBlurFileURL != "" {
			previewGif = first.GifBlurFileURL
		}
		if previewGif == "" && c.ThumbnailURL != "" {
			previewGif = c.ThumbnailURL
		}
		tagNames, _ := s.GetContentTagNames(c.ID)
		previewW, previewH := 0, 0
		if first.Width != nil && first.Height != nil && *first.Width > 0 && *first.Height > 0 {
			previewW, previewH = *first.Width, *first.Height
		}
		dur := c.Duration
		if dur == nil && first.Duration != nil && *first.Duration > 0 {
			dur = first.Duration
		}
		out = append(out, ListFeedItem{
			ID:             c.ID,
			Name:           c.Name,
			Description:    c.Description,
			Category:       string(c.Category),
			Price:          c.Price,
			PreviewGifURL:  previewGif,
			FirstFileID:    first.ID,
			PreviewWidth:   previewW,
			PreviewHeight:  previewH,
			Duration:       dur,
			Purchased:      purchased,
			Bookmarked:     bookmarked,
			CreatedAt:      c.CreatedAt.Format(time.RFC3339),
			AuthorUsername: authorUsername,
			Tags:           tagNames,
		})
	}
	return out, nil
}

// ListFeed 按分类分页列出 feed 内容（仅 ready），含首文件 gif 与当前用户是否已购买；freeOnly 为 true 时仅返回 price=0 的免费内容
func (s *contentService) ListFeed(category string, userID uint, limit, offset int, sortBy string, freeOnly bool) ([]ListFeedItem, error) {
	contents, err := s.contentRepo.ListFeedByCategory(category, limit, offset, sortBy, freeOnly)
	if err != nil {
		return nil, err
	}
	return s.buildListFeedItems(contents, userID)
}

// ListFeedByTag 按标签名列出 feed 内容（仅 ready）。支持带空格的标签名；先按 name 查，未找到再按 slug（空格转连字符）查。freeOnly 为 true 时仅返回 price=0 的免费内容
func (s *contentService) ListFeedByTag(tagName string, userID uint, limit, offset int, sortBy string, freeOnly bool) ([]ListFeedItem, error) {
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return nil, nil
	}
	tag, err := s.tagRepo.GetByName(tagName)
	if err != nil || tag == nil {
		// 按 name 未找到时，尝试按 slug 查找（与 ensureTagExists 中 slug 规则一致：空格转连字符、小写）
		slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))
		tag, err = s.tagRepo.GetBySlug(slug)
	}
	if err != nil || tag == nil {
		return []ListFeedItem{}, nil
	}
	contents, err := s.contentTagRepo.GetReadyContentsByTagID(tag.ID, limit, offset, sortBy, freeOnly)
	if err != nil {
		return nil, err
	}
	return s.buildListFeedItems(contents, userID)
}

// ListFeedSearch 模糊搜索 name/description 列出 feed（仅 ready），category 为空时搜全部；freeOnly 为 true 时仅返回 price=0 的免费内容
func (s *contentService) ListFeedSearch(q string, category string, userID uint, limit, offset int, sortBy string, freeOnly bool) ([]ListFeedItem, error) {
	contents, err := s.contentRepo.SearchReady(q, category, limit, offset, sortBy, freeOnly)
	if err != nil {
		return nil, err
	}
	return s.buildListFeedItems(contents, userID)
}

// GetAuthorForUserID 根据上传者 ID 返回作者用户名和头像 URL
func (s *contentService) GetAuthorForUserID(userID uint) (username string, avatar string) {
	if userID == 0 || s.userRepo == nil {
		return "", ""
	}
	if u, err := s.userRepo.GetByIDFromShard(userID, userID); err == nil && u != nil {
		if u.Username != nil && *u.Username != "" {
			return *u.Username, ""
		}
		if u.Email != "" {
			return u.Email, ""
		}
		return "", ""
	}
	if u, err := s.userRepo.GetByID(userID); err == nil && u != nil {
		if u.Username != nil && *u.Username != "" {
			return *u.Username, ""
		}
		if u.Email != "" {
			return u.Email, ""
		}
	}
	return "", ""
}

// ListPurchasedContent 返回当前用户已购买的内容列表（Library 页）
func (s *contentService) ListPurchasedContent(userID uint, limit, offset int) ([]ListFeedItem, error) {
	if userID == 0 {
		return nil, nil
	}
	purchases, err := s.purchaseRepo.GetByType(userID, entity.PurchaseTypeContent, limit*3, offset) // fetch extra for filtering
	if err != nil {
		return nil, err
	}
	seen := make(map[uint]bool)
	var contentIDs []uint
	for _, p := range purchases {
		if p.Status != entity.PurchaseStatusCompleted || p.ContentID == nil {
			continue
		}
		cid := *p.ContentID
		if seen[cid] {
			continue
		}
		seen[cid] = true
		contentIDs = append(contentIDs, cid)
		if len(contentIDs) >= limit {
			break
		}
	}
	out := make([]ListFeedItem, 0, len(contentIDs))
	for _, cid := range contentIDs {
		c, err := s.contentRepo.GetByID(cid)
		if err != nil || !c.IsReady() {
			continue
		}
		authorUsername := randomUsername()
		files, err := s.fileRepo.GetByContentID(c.ID)
		if err != nil || len(files) == 0 {
			tagNames, _ := s.GetContentTagNames(c.ID)
			out = append(out, ListFeedItem{
				ID:             c.ID,
				Name:           c.Name,
				Description:    c.Description,
				Category:       string(c.Category),
				Price:          c.Price,
				Duration:       c.Duration,
				CreatedAt:      c.CreatedAt.Format(time.RFC3339),
				AuthorUsername: authorUsername,
				Purchased:      true,
				Bookmarked:     true,
				Tags:           tagNames,
			})
			continue
		}
		first := files[0]
		bookmarked := true
		tagNames, _ := s.GetContentTagNames(c.ID)
		previewW, previewH := 0, 0
		if first.Width != nil && first.Height != nil && *first.Width > 0 && *first.Height > 0 {
			previewW, previewH = *first.Width, *first.Height
		}
		dur := c.Duration
		if dur == nil && first.Duration != nil && *first.Duration > 0 {
			dur = first.Duration
		}
		out = append(out, ListFeedItem{
			ID:             c.ID,
			Name:           c.Name,
			Description:    c.Description,
			Category:       string(c.Category),
			Price:          c.Price,
			PreviewGifURL:  first.GifFileURL,
			FirstFileID:    first.ID,
			PreviewWidth:   previewW,
			PreviewHeight:  previewH,
			Duration:       dur,
			Purchased:      true,
			Bookmarked:     bookmarked,
			CreatedAt:      c.CreatedAt.Format(time.RFC3339),
			AuthorUsername: authorUsername,
			Tags:           tagNames,
		})
	}
	return out, nil
}

// AddBookmark 添加书签
func (s *contentService) AddBookmark(userID, contentID uint) error {
	_, err := s.contentRepo.GetByID(contentID)
	if err != nil {
		return fmt.Errorf("content not found: %w", err)
	}
	exists, err := s.contentBookmarkRepo.Exists(userID, contentID)
	if err != nil {
		return err
	}
	if exists {
		return nil // already bookmarked
	}
	return s.contentBookmarkRepo.Create(&entity.ContentBookmark{UserID: userID, ContentID: contentID})
}

// RemoveBookmark 移除书签
func (s *contentService) RemoveBookmark(userID, contentID uint) error {
	return s.contentBookmarkRepo.Delete(userID, contentID)
}

// IsBookmarked 当前用户是否已书签该内容
func (s *contentService) IsBookmarked(userID, contentID uint) (bool, error) {
	return s.contentBookmarkRepo.Exists(userID, contentID)
}

// UserCanViewContent 用户是否可观看该内容（已购买或有效会员，或拥有 can_view_all 权限如 admin）
func (s *contentService) UserCanViewContent(userID uint, contentID uint) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	// 拥有 can_view_all 权限（仅 admin）可无需购买或会员直接观看全部内容
	hasViewAll, err := s.userPermissionRepo.HasPermission(userID, "can_view_all")
	if err == nil && hasViewAll {
		return true, nil
	}
	purchased, err := s.purchaseRepo.HasUserPurchasedContent(userID, contentID)
	if err != nil {
		return false, err
	}
	if purchased {
		return true, nil
	}
	membership, err := s.purchaseRepo.GetLatestActiveMembership(userID)
	if err != nil || membership == nil {
		return false, nil
	}
	return true, nil
}

// ListBookmarkedContent 返回当前用户书签的内容列表（Bookmarks 页），未购买项预取模糊 GIF 与主 feed 一致
func (s *contentService) ListBookmarkedContent(userID uint, limit, offset int) ([]ListFeedItem, error) {
	if userID == 0 {
		return nil, nil
	}
	contentIDs, err := s.contentBookmarkRepo.ListContentIDsByUserID(userID, limit, offset)
	if err != nil {
		return nil, err
	}
	contents := make([]entity.Content, 0, len(contentIDs))
	for _, cid := range contentIDs {
		c, err := s.contentRepo.GetByID(cid)
		if err != nil || !c.IsReady() {
			continue
		}
		contents = append(contents, *c)
	}
	return s.buildListFeedItems(contents, userID)
}

// GetGifPreview 返回 GIF 预览：可观看（已购买或有效会员）则流式返回正常 GIF，否则返回模糊 GIF
func (s *contentService) GetGifPreview(fileID uint, userID uint) (io.ReadCloser, string, error) {
	file, err := s.fileRepo.GetByID(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("file not found: %w", err)
	}
	content, err := s.contentRepo.GetByID(file.ContentID)
	if err != nil {
		return nil, "", fmt.Errorf("content not found: %w", err)
	}
	canView := false
	if userID > 0 {
		canView, _ = s.UserCanViewContent(userID, content.ID)
	}
	var s3Key string
	if canView {
		if file.GifFilePath == "" {
			return nil, "", fmt.Errorf("no gif for file %d", fileID)
		}
		s3Key = file.GifFilePath
	} else {
		if file.GifBlurFilePath == "" {
			return nil, "", fmt.Errorf("no blur gif for file %d", fileID)
		}
		s3Key = file.GifBlurFilePath
	}
	reader, contentType, err := s.s3Service.StreamFile("", s3Key)
	if err != nil {
		return nil, "", err
	}
	if contentType == "" {
		contentType = "image/gif"
	}
	return reader, contentType, nil
}
