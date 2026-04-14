// bulk_upload 从本地文件夹读取视频，按与 API 相同的 S3 路径与 Kafka 流程上传并转码。
// 使用方式（须在 backend 目录下）: go run ./cmd/bulk_upload -folder=./videos
// 需配置 .env: DB_*, AWS_*, KAFKA_BROKERS
package main

import (
	"backend/config"
	"backend/entity"
	"backend/repository"
	"backend/service"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const (
	adminUserID     = 1
	priceCredits    = 0
	categoryDark    = "dark"
	maxTagsFromName = 5

	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

// logError 输出红色错误日志
func logError(format string, args ...interface{}) {
	log.Printf(colorRed+format+colorReset, args...)
}

// logFatal 输出红色错误日志并退出
func logFatal(format string, args ...interface{}) {
	log.Printf(colorRed+format+colorReset, args...)
	os.Exit(1)
}

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
	localMode := flag.Bool("local", false, "本地模式：不上传原文件到 S3，直接使用本地路径走 gif/转码流水线（仅当消费者与本机同一台时有效）")
	workers := flag.Int("workers", 1, "并发上传/入队数（>1 时多文件同时处理，不显示单文件 S3 进度条）")
	flag.Parse()
	if *folder == "" {
		logFatal("请使用 -folder=./videos 指定视频文件夹")
	}
	if *workers < 1 {
		*workers = 1
	}
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found, using env vars")
	}

	db, err := config.InitDB()
	if err != nil {
		logFatal("数据库初始化失败: %v", err)
	}
	repos := config.InitRepositories(db)
	s3Svc, err := service.NewS3Service()
	if err != nil {
		logFatal("S3 初始化失败: %v", err)
	}
	kafkaSvc, err := service.NewKafkaService()
	if err != nil {
		logFatal("Kafka 初始化失败: %v", err)
	}

	// 在本进程内启动 Kafka 消费者，使上传的视频走完整流水线（GIF → 转码 → 上传 → 删除），无需另开 backend 服务
	videoSvc, err := service.NewVideoProcessingService()
	if err != nil {
		logError("Warning: 视频处理服务初始化失败（需 ffmpeg/ffprobe）: %v。仅上传入队，需由其他进程消费 Kafka 完成处理。", err)
	} else {
		consumer := service.NewVideoProcessorConsumer(
			repos.ContentFileRepo,
			repos.ContentRepo,
			s3Svc,
			videoSvc,
			kafkaSvc,
		)
		if err := consumer.Start(); err != nil {
			logError("Warning: Kafka 消费者启动失败: %v。需由其他进程消费 Kafka 完成处理。", err)
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
		ninjaChat:      service.NewNinjaChatClient(),
	}

	// 先清理之前遗留的 processing/pending 内容（多半是失败未完成的），再处理新上传
	if n, err := uploader.cleanupStuckContents(); err != nil {
		logFatal("清理遗留 processing/pending 内容失败: %v", err)
	} else if n > 0 {
		log.Printf("已清理 %d 条遗留的 processing/pending 内容", n)
	}

	contentIDs, err := uploader.run(*folder, *localMode, *workers)
	if err != nil {
		logFatal("批量上传失败: %v", err)
	}
	if len(contentIDs) == 0 {
		log.Println("未发现视频文件，退出")
		return
	}
	log.Printf("批量上传完成，共 %d 个视频入队。开始监控处理进度…", len(contentIDs))
	retriesLeft := maxRetries - 1
	for {
		uploader.monitorUntilDone(contentIDs)
		failedIDs := uploader.getFailedContentIDs(contentIDs)
		if len(failedIDs) == 0 || retriesLeft <= 0 {
			break
		}
		log.Printf("重试 %d 个失败项（剩余重试次数 %d）…", len(failedIDs), retriesLeft)
		if err := uploader.retryFailed(failedIDs); err != nil {
			logError("重试入队失败: %v", err)
		}
		retriesLeft--
	}
	log.Println("全部处理完成，退出")
}

type bulkUploader struct {
	contentRepo    repository.ContentRepository
	fileRepo       repository.ContentFileRepository
	tagRepo        repository.TagRepository
	contentTagRepo repository.ContentTagRepository
	s3Service      service.S3Service
	kafkaService   service.KafkaService
	ninjaChat      *service.NinjaChatClient
}

const (
	pollInterval   = 15 * time.Second
	reportInterval = 5 * 60 * time.Second // 详细报告打印间隔
	maxRetries     = 2                    // 单文件最多尝试次数（含首次）
	cleanupBatch   = 500                  // 清理时每批查询条数
	progressBarLen = 24                   // 进度条长度（字符）
)

// s3ProgressBar 返回一个进度回调，在终端打印 S3 上传进度条（bytesRead/totalBytes）。
func s3ProgressBar(totalBytes int64) func(bytesRead, total int64) {
	if totalBytes <= 0 {
		return nil
	}
	return func(bytesRead, total int64) {
		if total <= 0 {
			return
		}
		pct := int64(0)
		if total > 0 {
			pct = bytesRead * 100 / total
			if pct > 100 {
				pct = 100
			}
		}
		filled := int(float64(progressBarLen) * float64(bytesRead) / float64(total))
		if filled > progressBarLen {
			filled = progressBarLen
		}
		bar := strings.Repeat("=", filled) + ">" + strings.Repeat(" ", progressBarLen-filled)
		if filled == progressBarLen {
			bar = strings.Repeat("=", progressBarLen)
		}
		mbR := bytesRead / (1024 * 1024)
		mbT := total / (1024 * 1024)
		fmt.Fprintf(os.Stderr, "\r  S3 [%s] %3d%% (%d/%d MB)", bar, pct, mbR, mbT)
		if bytesRead >= total {
			fmt.Fprintln(os.Stderr)
		}
	}
}

// cleanupStuckContents 删除所有 status 为 pending 或 processing 的内容（及其 content_files、content_tags），返回删除条数。
func (u *bulkUploader) cleanupStuckContents() (int, error) {
	var toDelete []entity.Content
	for _, status := range []entity.ContentStatus{entity.ContentStatusPending, entity.ContentStatusProcessing} {
		for offset := 0; ; offset += cleanupBatch {
			batch, err := u.contentRepo.GetByStatus(status, cleanupBatch, offset)
			if err != nil {
				return 0, fmt.Errorf("查询 %s 内容: %w", status, err)
			}
			if len(batch) == 0 {
				break
			}
			toDelete = append(toDelete, batch...)
		}
	}
	if len(toDelete) == 0 {
		return 0, nil
	}
	log.Printf("发现 %d 条遗留的 processing/pending 内容，开始清理…", len(toDelete))
	deleted := 0
	for _, c := range toDelete {
		files, _ := u.fileRepo.GetByContentID(c.ID)
		for _, f := range files {
			if err := u.fileRepo.Delete(f.ID); err != nil {
				logError("清理 content_id=%d file_id=%d 失败: %v", c.ID, f.ID, err)
				continue
			}
		}
		_ = u.contentTagRepo.RemoveAllTagsFromContent(c.ID)
		if err := u.contentRepo.Delete(c.ID); err != nil {
			logError("删除 content_id=%d 失败: %v", c.ID, err)
			continue
		}
		deleted++
		log.Printf("  已删除 content_id=%d name=%s", c.ID, c.Name)
	}
	return deleted, nil
}

func (u *bulkUploader) run(folder string, localMode bool, workers int) ([]uint, error) {
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
	var videoPaths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := e.Name()
		ext := strings.ToLower(filepath.Ext(base))
		if _, ok := videoExts[ext]; !ok {
			continue
		}
		videoPaths = append(videoPaths, filepath.Join(folder, base))
	}
	total := len(videoPaths)
	if total == 0 {
		return nil, nil
	}
	modeDesc := "上传S3 -> 入队"
	if localMode {
		modeDesc = "本地路径入队（不传 S3）"
	}
	log.Printf("发现 %d 个视频文件，workers=%d，%s", total, workers, modeDesc)

	var contentIDs []uint
	var mu sync.Mutex
	if workers <= 1 {
		for i, path := range videoPaths {
			base := filepath.Base(path)
			ext := strings.ToLower(filepath.Ext(base))
			log.Printf("[%d/%d] %s: 创建内容 -> %s", i+1, total, base, modeDesc)
			cid, err := u.uploadOne(path, base, ext, localMode, true)
			if err != nil {
				log.Printf("  -> 跳过: %v", err)
				continue
			}
			contentIDs = append(contentIDs, cid)
			log.Printf("  -> 完成，content_id=%d", cid)
		}
		return contentIDs, nil
	}
	// 多 worker：并发入队，不显示单文件进度条
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for i, path := range videoPaths {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			base := filepath.Base(p)
			ext := strings.ToLower(filepath.Ext(base))
			cid, err := u.uploadOne(p, base, ext, localMode, false)
			mu.Lock()
			if err != nil {
				logError("[%d/%d] %s 跳过: %v", idx+1, total, base, err)
			} else {
				contentIDs = append(contentIDs, cid)
				log.Printf("[%d/%d] %s 完成 content_id=%d", idx+1, total, base, cid)
			}
			mu.Unlock()
		}(i, path)
	}
	wg.Wait()
	return contentIDs, nil
}

// monitorUntilDone 轮询直到本批所有内容的 status 为 ready 或 failed；每 reportInterval 打印详细报告
func (u *bulkUploader) monitorUntilDone(contentIDs []uint) {
	total := len(contentIDs)
	lastReady, lastFailed, lastProcessing := -1, -1, -1
	lastReport := time.Now()
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
		if time.Since(lastReport) >= reportInterval {
			u.printDetailedReport(contentIDs)
			lastReport = time.Now()
		}
		if ready+failed == total {
			return
		}
		time.Sleep(pollInterval)
	}
}

// printDetailedReport 打印本批每个内容的详细状态（content_id, name, status, file_stage, error）
func (u *bulkUploader) printDetailedReport(contentIDs []uint) {
	log.Println("---------- 详细上传报告 ----------")
	for _, cid := range contentIDs {
		c, err := u.contentRepo.GetByID(cid)
		if err != nil {
			logError("  [%d] 查询失败: %v", cid, err)
			continue
		}
		files, _ := u.fileRepo.GetByContentID(cid)
		fileStage := "-"
		fileErr := ""
		if len(files) > 0 {
			f := files[0]
			fileStage = string(f.Stage)
			if f.ProcessingError != "" {
				fileErr = f.ProcessingError
				if len(fileErr) > 80 {
					fileErr = fileErr[:77] + "..."
				}
			}
		}
		log.Printf("  [%d] %s | content=%s | file_stage=%s | %s", cid, c.Name, c.Status, fileStage, fileErr)
	}
	log.Println("----------------------------------")
}

// getFailedContentIDs 返回本批中状态为 failed 的内容 ID 列表
func (u *bulkUploader) getFailedContentIDs(contentIDs []uint) []uint {
	var out []uint
	for _, cid := range contentIDs {
		c, err := u.contentRepo.GetByID(cid)
		if err != nil {
			continue
		}
		if c.Status == entity.ContentStatusFailed {
			out = append(out, cid)
		}
	}
	return out
}

// retryFailed 将失败项清理并重新入队：重置文件为 uploaded、内容为 processing，重新发送 Kafka gif_generation（衍生资源已在 consumer 失败时清理）
func (u *bulkUploader) retryFailed(contentIDs []uint) error {
	for _, cid := range contentIDs {
		content, err := u.contentRepo.GetByID(cid)
		if err != nil {
			logError("重试 [%d]: 获取内容失败 %v", cid, err)
			continue
		}
		files, err := u.fileRepo.GetByContentID(cid)
		if err != nil || len(files) == 0 {
			logError("重试 [%d]: 无文件记录", cid)
			continue
		}
		for _, f := range files {
			f.GifFilePath = ""
			f.GifFileURL = ""
			f.GifFileSize = 0
			f.GifBlurFilePath = ""
			f.GifBlurFileURL = ""
			f.GifBlurFileSize = 0
			f.TranscodedFilePath = ""
			f.TranscodedFileURL = ""
			f.TranscodedFileSize = 0
			f.Stage = entity.StageUploaded
			f.ProcessingError = ""
			f.ProcessedAt = nil
			if err := u.fileRepo.Update(f); err != nil {
				logError("重试 [%d] file %d: 更新文件失败 %v", cid, f.ID, err)
				continue
			}
			content.Status = entity.ContentStatusProcessing
			if err := u.contentRepo.Update(content); err != nil {
				logError("重试 [%d]: 更新内容失败 %v", cid, err)
				continue
			}
			data := map[string]interface{}{
				"s3_key":   f.OriginalFilePath,
				"file_url": f.OriginalFileURL,
			}
			if strings.HasPrefix(f.OriginalFilePath, "local:") {
				data["local_path"] = strings.TrimPrefix(f.OriginalFilePath, "local:")
			}
			msg := &service.KafkaMessage{
				Type:      "gif_generation",
				ContentID: content.ID,
				FileID:    f.ID,
				Stage:     string(entity.StageUploaded),
				Data:      data,
			}
			if err := u.kafkaService.SendMessage(service.TopicGifGeneration, msg); err != nil {
				logError("重试 [%d]: 发送 Kafka 失败 %v", cid, err)
				continue
			}
			log.Printf("重试入队: content_id=%d, file_id=%d", cid, f.ID)
		}
	}
	return nil
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

// translateToEnglish 使用 LibreTranslate 将中文翻译为英文，失败或未配置时返回原串。
// 环境变量: LIBRETRANSLATE_URL（如 http://localhost:5000），不设则跳过翻译。
func translateToEnglish(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	url := os.Getenv("LIBRETRANSLATE_URL")
	if url == "" {
		return text
	}
	url = strings.TrimSuffix(url, "/") + "/translate"
	body, _ := json.Marshal(map[string]string{
		"q":      text,
		"source": "zh",
		"target": "en",
	})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return text
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// 可选依赖未启动时不刷屏红色错误
		log.Printf("LibreTranslate 不可用，跳过翻译: %v", err)
		return text
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return text
	}
	var out struct {
		TranslatedText string `json:"translatedText"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return text
	}
	out.TranslatedText = strings.TrimSpace(out.TranslatedText)
	if out.TranslatedText != "" {
		return out.TranslatedText
	}
	return text
}

const maxContentNameRunes = 255

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// stripLLMJSONFence 去掉模型可能返回的 ```json ... ``` 包裹。
func stripLLMJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "json") {
		s = strings.TrimSpace(s[4:])
	}
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// extractJSONObject 从文本中截取第一个花括号平衡的 JSON 对象（尊重字符串内的引号与转义），用于模型在 JSON 前后加了说明的情况。
func extractJSONObject(s string) (string, bool) {
	idx := strings.Index(s, "{")
	if idx < 0 {
		return "", false
	}
	depth := 0
	inString := false
	escape := false
	for i := idx; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			if c == '\\' {
				escape = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[idx : i+1], true
			}
		}
	}
	return "", false
}

func parseNinjaTitleDescriptionJSON(raw string) (title, description string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("empty ninjachat message")
	}
	raw = stripLLMJSONFence(raw)
	var candidates []string
	candidates = append(candidates, raw)
	if obj, ok := extractJSONObject(raw); ok {
		candidates = append(candidates, obj)
	}
	var out struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	for _, c := range candidates {
		if err := json.Unmarshal([]byte(c), &out); err != nil {
			continue
		}
		t := strings.TrimSpace(out.Title)
		if t == "" {
			continue
		}
		return truncateRunes(t, maxContentNameRunes), strings.TrimSpace(out.Description), nil
	}
	return "", "", fmt.Errorf("parse ninja json: invalid or truncated response")
}

// ninjaEroticTitleAndDescription 用 NinjaChat 根据文件名提示生成英文标题与成人向描述；失败时返回空串与错误，由调用方回退。
func ninjaEroticTitleAndDescription(ctx context.Context, client *service.NinjaChatClient, baseName, parsedName string) (title, description string, err error) {
	if client == nil {
		return "", "", fmt.Errorf("ninjachat not configured")
	}
	stem := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	stem = strings.TrimSpace(stem)
	prompt := fmt.Sprintf(`Adult erotic video catalog (18+ only). Filename stem: %q. Parsed guess: %q.

Return ONLY valid JSON, one line if possible, no markdown:
{"title":"English title under 90 chars","description":"Max 2 short sensual sentences in English; adults-only; no minors or illegal themes."}`, stem, parsedName)
	const maxTok = 1024
	raw, err := client.Chat(ctx, prompt, maxTok)
	if err != nil {
		return "", "", err
	}
	title, description, err = parseNinjaTitleDescriptionJSON(raw)
	if err != nil {
		// 常见原因：max_tokens 截断；用更短指令重试一次
		retryPrompt := fmt.Sprintf(`JSON only: {"title":"erotic English title from %q","description":"one teaser sentence"}`,
			truncateRunes(stem, 120))
		raw2, err2 := client.Chat(ctx, retryPrompt, 512)
		if err2 != nil {
			return "", "", err
		}
		return parseNinjaTitleDescriptionJSON(raw2)
	}
	return title, description, nil
}

// nameForDescription 将名称用 LibreTranslate 从中文译为英文后返回，用于 Description。
func nameForDescription(baseName string) string {
	namePart := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	namePart = strings.TrimSpace(namePart)
	if namePart == "" {
		return baseName
	}
	translated := translateToEnglish(namePart)
	if translated == namePart {
		return baseName
	}
	ext := filepath.Ext(baseName)
	if ext != "" {
		return translated + ext
	}
	return translated
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

func (u *bulkUploader) uploadOne(localPath, baseName, ext string, localMode, showProgress bool) (contentID uint, err error) {
	name, tagNames := parseNameAndTags(baseName)
	contentType := videoExts[ext]

	desc := fmt.Sprintf("Uploaded from %s", nameForDescription(baseName))
	if u.ninjaChat != nil {
		llmCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		llmTitle, llmDesc, llmErr := ninjaEroticTitleAndDescription(llmCtx, u.ninjaChat, baseName, name)
		cancel()
		if llmErr != nil {
			logError("NinjaChat 生成标题/描述失败 (%s): %v", baseName, llmErr)
		} else {
			name = llmTitle
			if strings.TrimSpace(llmDesc) != "" {
				desc = llmDesc
			}
		}
	}

	content := &entity.Content{
		Name:        name,
		Description: desc,
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
			logError("创建标签 %s 失败: %v", tn, err)
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

	stat, err := os.Stat(localPath)
	if err != nil {
		return 0, fmt.Errorf("stat 文件: %w", err)
	}
	fileSize := stat.Size()

	var s3Key, fileURL string
	if localMode {
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			absPath = localPath
		}
		s3Key = "local:" + absPath
		fileURL = ""
	} else {
		timestamp := time.Now().Unix()
		s3Key = fmt.Sprintf("videos/%d/%d_%s", adminUserID, timestamp, baseName)
		var onProgress func(bytesRead, total int64)
		if showProgress && fileSize > 0 {
			onProgress = s3ProgressBar(fileSize)
		}
		fileURL, err = u.s3Service.UploadFileFromPathWithProgress("", s3Key, localPath, contentType, onProgress)
		if err != nil {
			return 0, fmt.Errorf("上传 S3: %w", err)
		}
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

	data := map[string]interface{}{
		"s3_key":   s3Key,
		"file_url": fileURL,
	}
	if localMode {
		absPath, _ := filepath.Abs(localPath)
		data["local_path"] = absPath
	}
	msg := &service.KafkaMessage{
		Type:      "gif_generation",
		ContentID: content.ID,
		FileID:    contentFile.ID,
		Stage:     string(entity.StageUploaded),
		Data:      data,
	}
	if err := u.kafkaService.SendMessage(service.TopicGifGeneration, msg); err != nil {
		return 0, fmt.Errorf("发送 Kafka: %w", err)
	}
	return contentID, nil
}
