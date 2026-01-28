package service

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
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

	// GetContentAnalytics 获取内容分析数据
	GetContentAnalytics(contentID uint, startDate, endDate string) (map[string]interface{}, error)
}

// contentService 内容服务实现
type contentService struct {
	contentRepo    repository.ContentRepository
	fileRepo       repository.ContentFileRepository
	tagRepo        repository.TagRepository
	contentTagRepo repository.ContentTagRepository
	analyticsRepo  repository.AnalyticsRepository
	s3Service      S3Service
	kafkaService   KafkaService
}

// NewContentService 创建内容服务实例
func NewContentService(
	contentRepo repository.ContentRepository,
	fileRepo repository.ContentFileRepository,
	tagRepo repository.TagRepository,
	contentTagRepo repository.ContentTagRepository,
	analyticsRepo repository.AnalyticsRepository,
	s3Service S3Service,
	kafkaService KafkaService,
) ContentService {
	return &contentService{
		contentRepo:    contentRepo,
		fileRepo:       fileRepo,
		tagRepo:        tagRepo,
		contentTagRepo: contentTagRepo,
		analyticsRepo:  analyticsRepo,
		s3Service:      s3Service,
		kafkaService:   kafkaService,
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

// UploadVideos 批量上传视频（最多10个），每个视频有独立的元数据
func (s *contentService) UploadVideos(userID uint, files []*multipart.FileHeader, videosMetadata []VideoMetadata) ([]*entity.ContentFile, error) {
	// 验证文件数量
	if len(files) == 0 {
		return nil, errors.New("no files provided")
	}
	if len(files) > 10 {
		return nil, errors.New("maximum 10 files allowed")
	}

	// 验证元数据数量与文件数量匹配
	if len(videosMetadata) != len(files) {
		return nil, fmt.Errorf("number of video metadata (%d) does not match number of files (%d)", len(videosMetadata), len(files))
	}

	// 处理每个文件，为每个视频创建独立的 Content 记录
	contentFiles := make([]*entity.ContentFile, 0, len(files))

	for i, fileHeader := range files {
		metadata := videosMetadata[i]

		// 验证元数据
		if strings.TrimSpace(metadata.Name) == "" {
			return nil, fmt.Errorf("video %d: name is required", i+1)
		}

		// 确定分类（默认为 light）
		category := entity.ContentCategoryLight
		if metadata.Category == "dark" {
			category = entity.ContentCategoryDark
		}

		// 创建内容记录（每个视频独立）
		content := &entity.Content{
			Name:        strings.TrimSpace(metadata.Name),
			Description: strings.TrimSpace(metadata.Description),
			Type:        entity.ContentTypeVideo,
			Status:      entity.ContentStatusPending,
			UploadedBy:  userID,
			Price:       metadata.Price, // 设置价格
			Category:    category,       // 设置分类
		}

		if err := s.contentRepo.Create(content); err != nil {
			return nil, fmt.Errorf("failed to create content for video %d: %w", i+1, err)
		}

		// 处理标签：查找或创建标签，并关联到内容
		if len(metadata.Tags) > 0 {
			tagIDs := make([]uint, 0, len(metadata.Tags))
			for _, tagName := range metadata.Tags {
				tagID, err := s.ensureTagExists(tagName)
				if err != nil {
					return nil, fmt.Errorf("failed to process tag '%s' for video %d: %w", tagName, i+1, err)
				}
				if tagID > 0 {
					tagIDs = append(tagIDs, tagID)
				}
			}

			// 批量添加标签到内容
			if len(tagIDs) > 0 {
				if err := s.contentTagRepo.AddTagsToContent(content.ID, tagIDs); err != nil {
					return nil, fmt.Errorf("failed to add tags to content %d: %w", content.ID, err)
				}
			}
		}

		// 打开文件
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", fileHeader.Filename, err)
		}

		// 生成S3路径
		timestamp := time.Now().Unix()
		s3Key := fmt.Sprintf("videos/%d/%d_%s", userID, timestamp, filepath.Base(fileHeader.Filename))

		// 上传到S3
		fileURL, err := s.s3Service.UploadFile("", s3Key, file, fileHeader.Header.Get("Content-Type"))
		file.Close() // 立即关闭文件
		if err != nil {
			return nil, fmt.Errorf("failed to upload file to S3: %w", err)
		}

		// 获取文件大小
		fileSize := fileHeader.Size

		// 创建文件记录
		contentFile := &entity.ContentFile{
			ContentID:        content.ID,
			FileName:         fileHeader.Filename,
			FileIndex:        0, // 每个内容只有一个文件，所以索引为0
			OriginalFilePath: s3Key,
			OriginalFileURL:  fileURL,
			OriginalFileSize: fileSize,
			MimeType:         fileHeader.Header.Get("Content-Type"),
			Stage:            entity.StageUploaded,
		}

		if err := s.fileRepo.Create(contentFile); err != nil {
			return nil, fmt.Errorf("failed to create file record: %w", err)
		}

		contentFiles = append(contentFiles, contentFile)

		// 更新内容状态为处理中
		content.Status = entity.ContentStatusProcessing
		if err := s.contentRepo.Update(content); err != nil {
			return nil, fmt.Errorf("failed to update content status: %w", err)
		}

		// 发送Kafka消息触发GIF生成
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
			// 记录错误但不中断流程
			fmt.Printf("Failed to send Kafka message for file %d: %v\n", contentFile.ID, err)
		}
	}

	return contentFiles, nil
}

// GetContent 获取内容
func (s *contentService) GetContent(contentID uint) (*entity.Content, error) {
	return s.contentRepo.GetByID(contentID)
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
// .ts 分片文件与 .m3u8 文件在同一目录下
func (s *contentService) GetSegmentFilePath(m3u8Path string, segmentName string) string {
	// 移除 .m3u8 文件名，获取目录路径
	dir := filepath.Dir(m3u8Path)
	// 构建分片文件的完整路径（使用 filepath.Join 确保路径正确）
	// 注意：segmentName 可能包含查询参数，需要提取实际文件名
	actualSegmentName := segmentName
	if idx := strings.Index(segmentName, "?"); idx != -1 {
		actualSegmentName = segmentName[:idx]
	}
	return filepath.Join(dir, actualSegmentName)
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
