// bulk_upload 从本地文件夹读取视频，按与 API 相同的 S3 路径与 Kafka 流程上传并转码。
// 使用方式: go run backend/cmd/bulk_upload/main.go -folder=./videos
// 需配置 .env: DB_*, AWS_*, KAFKA_BROKERS
package main

import (
	"backend/config"
	"backend/entity"
	"backend/repository"
	"backend/service"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const (
	adminUserID     = 7
	priceCredits    = 10
	categoryDark    = "dark"
	maxTagsFromName = 5
)

var (
	videoExts = map[string]string{
		".mp4":  "video/mp4",
		".mov":  "video/quicktime",
		".webm": "video/webm",
		".avi":  "video/x-msvideo",
	}
)

func main() {
	folder := flag.String("folder", "", "本地视频文件夹路径（必填）")
	flag.Parse()
	if *folder == "" {
		log.Fatal("请使用 -folder=./videos 指定视频文件夹")
	}
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found, using env vars")
	}

	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	repos := config.InitRepositories(db)
	s3Svc, err := service.NewS3Service()
	if err != nil {
		log.Fatalf("S3 初始化失败: %v", err)
	}
	kafkaSvc, err := service.NewKafkaService()
	if err != nil {
		log.Fatalf("Kafka 初始化失败: %v", err)
	}

	// 在本进程内启动 Kafka 消费者，使上传的视频走完整流水线（GIF → 转码 → 上传 → 删除），无需另开 backend 服务
	videoSvc, err := service.NewVideoProcessingService()
	if err != nil {
		log.Printf("Warning: 视频处理服务初始化失败（需 ffmpeg/ffprobe）: %v。仅上传入队，需由其他进程消费 Kafka 完成处理。", err)
	} else {
		consumer := service.NewVideoProcessorConsumer(
			repos.ContentFileRepo,
			repos.ContentRepo,
			s3Svc,
			videoSvc,
			kafkaSvc,
		)
		if err := consumer.Start(); err != nil {
			log.Printf("Warning: Kafka 消费者启动失败: %v。需由其他进程消费 Kafka 完成处理。", err)
		} else {
			log.Println("Kafka 消费者已启动，本进程将消费 gif-generation / video-transcode / transcode-upload / original-delete")
		}
	}

	uploader := &bulkUploader{
		contentRepo:    repos.ContentRepo,
		fileRepo:       repos.ContentFileRepo,
		tagRepo:        repos.TagRepo,
		contentTagRepo: repos.ContentTagRepo,
		s3Service:      s3Svc,
		kafkaService:   kafkaSvc,
	}

	contentIDs, err := uploader.run(*folder)
	if err != nil {
		log.Fatalf("批量上传失败: %v", err)
	}
	if len(contentIDs) == 0 {
		log.Println("未发现视频文件，退出")
		return
	}
	log.Printf("批量上传完成，共 %d 个视频入队。开始监控处理进度…", len(contentIDs))
	uploader.monitorUntilDone(contentIDs)
	log.Println("全部处理完成，退出")
}

type bulkUploader struct {
	contentRepo    repository.ContentRepository
	fileRepo       repository.ContentFileRepository
	tagRepo        repository.TagRepository
	contentTagRepo repository.ContentTagRepository
	s3Service      service.S3Service
	kafkaService   service.KafkaService
}

const pollInterval = 15 * time.Second

func (u *bulkUploader) run(folder string) ([]uint, error) {
	// Windows: 未加引号时 shell 会吃掉反斜杠，导致 C:\Users\... 变成 C:Users...，故统一用正斜杠再转成本地路径
	folder = filepath.FromSlash(folder)
	abs, _ := filepath.Abs(folder)
	info, err := os.Stat(folder)
	if err != nil {
		if os.IsNotExist(err) {
			hint := "请先创建该文件夹并放入视频，或使用 -folder= 指定已有视频目录。"
			if len(folder) > 1 && folder[1] == ':' && !strings.Contains(folder[2:], string(filepath.Separator)) {
				hint = "Windows 下请用引号包裹路径，例如 -folder=\"C:\\Users\\...\\videos\"，或使用正斜杠 -folder=C:/Users/.../videos"
			}
			return nil, fmt.Errorf("文件夹不存在: %s（绝对路径: %s）。%s", folder, abs, hint)
		}
		return nil, fmt.Errorf("读取文件夹 %s: %w", folder, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", folder)
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("读取文件夹 %s: %w", folder, err)
	}
	var contentIDs []uint
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := e.Name()
		ext := strings.ToLower(filepath.Ext(base))
		if _, ok := videoExts[ext]; !ok {
			continue
		}
		path := filepath.Join(folder, base)
		cid, err := u.uploadOne(path, base, ext)
		if err != nil {
			log.Printf("跳过 %s: %v", base, err)
			continue
		}
		contentIDs = append(contentIDs, cid)
		log.Printf("已入队: %s", base)
	}
	return contentIDs, nil
}

// monitorUntilDone 轮询直到本批所有内容的 status 为 ready 或 failed
func (u *bulkUploader) monitorUntilDone(contentIDs []uint) {
	total := len(contentIDs)
	lastReady, lastFailed, lastProcessing := -1, -1, -1
	for {
		ready, processing, failed := 0, 0, 0
		for _, cid := range contentIDs {
			c, err := u.contentRepo.GetByID(cid)
			if err != nil {
				failed++
				continue
			}
			switch c.Status {
			case entity.ContentStatusReady:
				ready++
			case entity.ContentStatusFailed:
				failed++
			default:
				processing++
			}
		}
		if ready != lastReady || failed != lastFailed || processing != lastProcessing {
			log.Printf("[监控] ready=%d, processing=%d, failed=%d (共 %d)", ready, processing, failed, total)
			lastReady, lastFailed, lastProcessing = ready, failed, processing
		}
		if ready+failed == total {
			return
		}
		time.Sleep(pollInterval)
	}
}

// parseNameAndTags 从文件名解析显示名和标签（如 summer_beach_fun.mp4 -> "Summer Beach Fun", ["summer","beach","fun"]）
func parseNameAndTags(baseName string) (name string, tags []string) {
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		name = "Untitled"
		return name, nil
	}
	// 用 _ 或 - 或空格 拆成片段
	repl := strings.NewReplacer("_", " ", "-", " ")
	parts := strings.Fields(repl.Replace(baseName))
	if len(parts) == 0 {
		name = "Untitled"
		return name, nil
	}
	// 首字母大写的显示名
	nameParts := make([]string, len(parts))
	for i, p := range parts {
		if len(p) > 0 {
			nameParts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		} else {
			nameParts[i] = p
		}
	}
	name = strings.Join(nameParts, " ")
	// 标签：取小写、去重、最多 maxTagsFromName 个
	seen := make(map[string]bool)
	for _, p := range parts {
		t := strings.ToLower(strings.TrimSpace(p))
		if t == "" || len(t) > 50 {
			continue
		}
		if !seen[t] {
			seen[t] = true
			tags = append(tags, t)
			if len(tags) >= maxTagsFromName {
				break
			}
		}
	}
	return name, tags
}

func (u *bulkUploader) ensureTagExists(tagName string) (uint, error) {
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return 0, fmt.Errorf("tag name cannot be empty")
	}
	tag, err := u.tagRepo.GetByName(tagName)
	if err == nil {
		return tag.ID, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("query tag: %w", err)
	}
	slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))
	originalSlug := slug
	for i := 0; ; i++ {
		_, err := u.tagRepo.GetBySlug(slug)
		if err == gorm.ErrRecordNotFound {
			break
		}
		if err != nil {
			return 0, err
		}
		slug = fmt.Sprintf("%s-%d", originalSlug, i+1)
	}
	newTag := &entity.Tag{Name: tagName, Slug: slug}
	if err := u.tagRepo.Create(newTag); err != nil {
		return 0, err
	}
	return newTag.ID, nil
}

func (u *bulkUploader) uploadOne(localPath, baseName, ext string) (contentID uint, err error) {
	name, tagNames := parseNameAndTags(baseName)
	contentType := videoExts[ext]

	content := &entity.Content{
		Name:        name,
		Description: fmt.Sprintf("Uploaded from %s", baseName),
		Type:        entity.ContentTypeVideo,
		Status:      entity.ContentStatusPending,
		UploadedBy:  adminUserID,
		Price:       priceCredits,
		Category:    entity.ContentCategoryDark,
	}
	if err := u.contentRepo.Create(content); err != nil {
		return 0, fmt.Errorf("创建内容: %w", err)
	}
	contentID = content.ID

	var tagIDs []uint
	for _, tn := range tagNames {
		id, err := u.ensureTagExists(tn)
		if err != nil {
			log.Printf("创建标签 %s 失败: %v", tn, err)
			continue
		}
		if id > 0 {
			tagIDs = append(tagIDs, id)
		}
	}
	if len(tagIDs) > 0 {
		if err := u.contentTagRepo.AddTagsToContent(content.ID, tagIDs); err != nil {
			return 0, fmt.Errorf("添加标签: %w", err)
		}
	}

	f, err := os.Open(localPath)
	if err != nil {
		return 0, fmt.Errorf("打开文件: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return 0, err
	}
	fileSize := stat.Size()

	timestamp := time.Now().Unix()
	s3Key := fmt.Sprintf("videos/%d/%d_%s", adminUserID, timestamp, baseName)
	fileURL, err := u.s3Service.UploadFile("", s3Key, f, contentType)
	if err != nil {
		return 0, fmt.Errorf("上传 S3: %w", err)
	}

	contentFile := &entity.ContentFile{
		ContentID:        content.ID,
		FileName:         baseName,
		FileIndex:        0,
		OriginalFilePath: s3Key,
		OriginalFileURL:  fileURL,
		OriginalFileSize: fileSize,
		MimeType:         contentType,
		Stage:            entity.StageUploaded,
	}
	if err := u.fileRepo.Create(contentFile); err != nil {
		return 0, fmt.Errorf("创建文件记录: %w", err)
	}

	content.Status = entity.ContentStatusProcessing
	if err := u.contentRepo.Update(content); err != nil {
		return 0, fmt.Errorf("更新内容状态: %w", err)
	}

	msg := &service.KafkaMessage{
		Type:      "gif_generation",
		ContentID: content.ID,
		FileID:    contentFile.ID,
		Stage:     string(entity.StageUploaded),
		Data: map[string]interface{}{
			"s3_key":   s3Key,
			"file_url": fileURL,
		},
	}
	if err := u.kafkaService.SendMessage(service.TopicGifGeneration, msg); err != nil {
		return 0, fmt.Errorf("发送 Kafka: %w", err)
	}
	// Kafka 消费者会依次：生成 GIF -> 设置 content.ThumbnailURL = gifURL -> 转码 -> 上传转码 -> 删除原文件 -> 内容 status=ready
	return contentID, nil
}
