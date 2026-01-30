package service

import (
	"backend/entity"
	"backend/repository"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VideoProcessorConsumer 视频处理消费者
type VideoProcessorConsumer struct {
	fileRepo               repository.ContentFileRepository
	contentRepo            repository.ContentRepository
	s3Service              S3Service
	videoProcessingService VideoProcessingService
	kafkaService           KafkaService
	tempDir                string
}

// NewVideoProcessorConsumer 创建视频处理消费者
func NewVideoProcessorConsumer(
	fileRepo repository.ContentFileRepository,
	contentRepo repository.ContentRepository,
	s3Service S3Service,
	videoProcessingService VideoProcessingService,
	kafkaService KafkaService,
) *VideoProcessorConsumer {
	tempDir := os.Getenv("TEMP_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	return &VideoProcessorConsumer{
		fileRepo:               fileRepo,
		contentRepo:            contentRepo,
		s3Service:              s3Service,
		videoProcessingService: videoProcessingService,
		kafkaService:           kafkaService,
		tempDir:                tempDir,
	}
}

// Start 启动所有消费者
func (v *VideoProcessorConsumer) Start() error {
	// 启动GIF生成消费者
	if err := v.kafkaService.StartConsumer(TopicGifGeneration, v.handleGifGeneration); err != nil {
		return fmt.Errorf("failed to start GIF generation consumer: %w", err)
	}

	// 启动视频转码消费者
	if err := v.kafkaService.StartConsumer(TopicVideoTranscode, v.handleVideoTranscode); err != nil {
		return fmt.Errorf("failed to start video transcode consumer: %w", err)
	}

	// 启动转码文件上传消费者
	if err := v.kafkaService.StartConsumer(TopicTranscodeUpload, v.handleTranscodeUpload); err != nil {
		return fmt.Errorf("failed to start transcode upload consumer: %w", err)
	}

	// 启动原始文件删除消费者
	if err := v.kafkaService.StartConsumer(TopicOriginalDelete, v.handleOriginalDelete); err != nil {
		return fmt.Errorf("failed to start original delete consumer: %w", err)
	}

	return nil
}

// handleGifGeneration 处理GIF生成任务
func (v *VideoProcessorConsumer) handleGifGeneration(msg *KafkaMessage) error {
	fileID := msg.FileID

	// 获取文件记录
	file, err := v.fileRepo.GetByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 非预期阶段时 ack 消息，避免重复重试（可能已失败或已进入下一阶段）
	if !file.CanGenerateGif() {
		return nil
	}

	// 下载原始视频到临时目录
	localVideoPath := filepath.Join(v.tempDir, fmt.Sprintf("video_%d_%d.mp4", fileID, time.Now().Unix()))
	if err := v.s3Service.DownloadFileToPath("", file.OriginalFilePath, localVideoPath); err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to download video: %v", err))
		return err
	}
	defer func() { _ = os.Remove(localVideoPath) }()

	// 获取视频信息以使用整段视频时长生成GIF（帧从全片选取）
	videoInfo, err := v.videoProcessingService.GetVideoInfo(localVideoPath)
	if err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to get video info: %v", err))
		return err
	}
	duration := videoInfo.Duration
	if duration <= 0 {
		duration = 5.0
	}
	const gifFrameLimit = 10

	localGifPath := filepath.Join(v.tempDir, fmt.Sprintf("gif_%d_%d.gif", fileID, time.Now().Unix()))

	if err := v.videoProcessingService.GenerateGifSampled(localVideoPath, localGifPath, duration, gifFrameLimit); err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to generate GIF: %v", err))
		return err
	}
	defer func() { _ = os.Remove(localGifPath) }()

	// 上传前获取GIF文件大小（上传后 S3 服务会删除本地文件）
	gifFileSize := int64(0)
	if gifFileInfo, err := os.Stat(localGifPath); err == nil {
		gifFileSize = gifFileInfo.Size()
	}

	// 上传GIF到S3（成功或失败后本地文件都会在 defer 中尝试删除）
	gifS3Key := fmt.Sprintf("gifs/%d/%d_%d.gif", file.ContentID, fileID, time.Now().Unix())
	gifURL, err := v.s3Service.UploadFileFromPath("", gifS3Key, localGifPath, "image/gif")
	if err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to upload GIF: %v", err))
		return err
	}

	// 更新文件记录（失败时删除已上传的 GIF）
	file.GifFilePath = gifS3Key
	file.GifFileURL = gifURL
	file.GifFileSize = gifFileSize
	file.Stage = entity.StageGifGenerated
	if err := v.fileRepo.Update(file); err != nil {
		_ = v.s3Service.DeleteFile("", gifS3Key)
		return fmt.Errorf("failed to update file: %w", err)
	}

	// 将生成的 GIF 设为该条内容的缩略图（若尚未设置则用首个文件的 GIF；多文件时首个处理完的为准）
	content, err := v.contentRepo.GetByID(file.ContentID)
	if err == nil && content.ThumbnailURL == "" {
		content.ThumbnailURL = gifURL
		_ = v.contentRepo.Update(content)
	}

	// 发送转码任务消息（失败时删除已上传的 GIF）
	transcodeMsg := &KafkaMessage{
		Type:      "video_transcode",
		ContentID: file.ContentID,
		FileID:    fileID,
		Stage:     string(entity.StageGifGenerated),
		Data: map[string]interface{}{
			"s3_key": file.OriginalFilePath,
		},
	}
	if err := v.kafkaService.SendMessage(TopicVideoTranscode, transcodeMsg); err != nil {
		_ = v.s3Service.DeleteFile("", gifS3Key)
		return fmt.Errorf("failed to send transcode message: %w", err)
	}
	return nil
}

// handleVideoTranscode 处理视频转码任务
func (v *VideoProcessorConsumer) handleVideoTranscode(msg *KafkaMessage) error {
	fileID := msg.FileID

	// 获取文件记录
	file, err := v.fileRepo.GetByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 非预期阶段时 ack 消息，避免重复重试
	if !file.CanTranscode() {
		return nil
	}

	// 下载原始视频到临时目录
	localVideoPath := filepath.Join(v.tempDir, fmt.Sprintf("video_%d_%d.mp4", fileID, time.Now().Unix()))
	if err := v.s3Service.DownloadFileToPath("", file.OriginalFilePath, localVideoPath); err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to download video: %v", err))
		return err
	}
	defer func() { _ = os.Remove(localVideoPath) }()

	// 转码视频为HLS格式
	outputDir := filepath.Join(v.tempDir, fmt.Sprintf("hls_%d_%d", fileID, time.Now().Unix()))
	outputName := fmt.Sprintf("playlist_%d", fileID)

	transcodeOptions := &TranscodeOptions{
		Codec:      "libx264",
		Bitrate:    "2000k",
		Resolution: "1920x1080", // 转码到 1080p 分辨率
		Quality:    "medium",    // x264 preset（非 profile）：medium/fast/slow 等
		Format:     "hls",
	}

	hlsOutput, err := v.videoProcessingService.TranscodeVideo(localVideoPath, outputDir, outputName, transcodeOptions)
	if err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to transcode video to HLS: %v", err))
		return err
	}
	defer func() {
		// 无论上传成功或失败，最终都清理本地 m3u8/ts 与目录
		_ = os.Remove(hlsOutput.PlaylistPath)
		for _, segmentPath := range hlsOutput.SegmentPaths {
			_ = os.Remove(segmentPath)
		}
		_ = os.RemoveAll(outputDir)
	}()

	// 获取视频信息（从第一个分片或原始视频）
	if len(hlsOutput.SegmentPaths) > 0 {
		videoInfo, err := v.videoProcessingService.GetVideoInfo(localVideoPath)
		if err == nil {
			file.Width = &videoInfo.Width
			file.Height = &videoInfo.Height
			file.Duration = &videoInfo.Duration
		}
	}

	// 上传 .m3u8 播放列表文件到S3临时位置（失败时 defer 会清理本地 playlist/ts 和 outputDir）
	tempPlaylistS3Key := fmt.Sprintf("temp/transcoded/%d/%d_%d.m3u8", file.ContentID, fileID, time.Now().Unix())
	tempPlaylistURL, err := v.s3Service.UploadFileFromPath("", tempPlaylistS3Key, hlsOutput.PlaylistPath, "application/vnd.apple.mpegurl")
	if err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to upload playlist file to temp: %v", err))
		return err
	}
	_ = tempPlaylistURL

	// 上传所有 .ts 分片文件到S3临时位置（任一失败时删除已上传的 temp 并返回）
	tempSegmentS3Keys := make([]string, 0, len(hlsOutput.SegmentPaths))
	for i, segmentPath := range hlsOutput.SegmentPaths {
		segmentName := filepath.Base(segmentPath)
		tempSegmentS3Key := fmt.Sprintf("temp/transcoded/%d/%d_%d_%s", file.ContentID, fileID, time.Now().Unix(), segmentName)
		_, err := v.s3Service.UploadFileFromPath("", tempSegmentS3Key, segmentPath, "video/mp2t")
		if err != nil {
			v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to upload segment file %d to temp: %v", i, err))
			v.deleteS3Keys("", append([]string{tempPlaylistS3Key}, tempSegmentS3Keys...)...)
			return err
		}
		tempSegmentS3Keys = append(tempSegmentS3Keys, tempSegmentS3Key)
	}

	// 更新文件记录（失败时删除已上传的 temp）
	file.Stage = entity.StageTranscoded
	if err := v.fileRepo.Update(file); err != nil {
		v.deleteS3Keys("", append([]string{tempPlaylistS3Key}, tempSegmentS3Keys...)...)
		return fmt.Errorf("failed to update file: %w", err)
	}

	// 发送转码文件上传任务消息（失败时删除已上传的 temp）
	uploadMsg := &KafkaMessage{
		Type:      "transcode_upload",
		ContentID: file.ContentID,
		FileID:    fileID,
		Stage:     string(entity.StageTranscoded),
		Data: map[string]interface{}{
			"temp_playlist_s3_key": tempPlaylistS3Key,
			"temp_segment_s3_keys": tempSegmentS3Keys,
			"output_name":          outputName,
		},
	}
	if err := v.kafkaService.SendMessage(TopicTranscodeUpload, uploadMsg); err != nil {
		v.deleteS3Keys("", append([]string{tempPlaylistS3Key}, tempSegmentS3Keys...)...)
		return fmt.Errorf("failed to send upload message: %w", err)
	}
	return nil
}

// handleTranscodeUpload 处理转码文件上传任务（HLS格式）
func (v *VideoProcessorConsumer) handleTranscodeUpload(msg *KafkaMessage) error {
	fileID := msg.FileID

	// 获取文件记录
	file, err := v.fileRepo.GetByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 非预期阶段时 ack 消息，避免重复重试
	if !file.CanUploadTranscoded() {
		return nil
	}

	// 从消息中获取临时S3路径
	tempPlaylistS3Key, ok := msg.Data["temp_playlist_s3_key"].(string)
	if !ok || tempPlaylistS3Key == "" {
		return fmt.Errorf("temp playlist S3 key not found in message")
	}

	tempSegmentS3Keys, ok := msg.Data["temp_segment_s3_keys"].([]interface{})
	if !ok {
		return fmt.Errorf("temp segment S3 keys not found in message")
	}

	outputName, _ := msg.Data["output_name"].(string)
	if outputName == "" {
		outputName = fmt.Sprintf("playlist_%d", fileID)
	}

	// 下载 .m3u8 播放列表文件到本地
	localPlaylistPath := filepath.Join(v.tempDir, fmt.Sprintf("playlist_%d_%d.m3u8", fileID, time.Now().Unix()))
	if err := v.s3Service.DownloadFileToPath("", tempPlaylistS3Key, localPlaylistPath); err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to download playlist file: %v", err))
		return err
	}
	defer func() { _ = os.Remove(localPlaylistPath) }()

	// 下载所有 .ts 分片文件到本地（任一失败时已注册的 defers 会清理已下载的文件）
	localSegmentPaths := make([]string, 0, len(tempSegmentS3Keys))
	for i, tempSegmentS3Key := range tempSegmentS3Keys {
		segmentKey, ok := tempSegmentS3Key.(string)
		if !ok {
			continue
		}
		localSegmentPath := filepath.Join(v.tempDir, fmt.Sprintf("segment_%d_%d_%d.ts", fileID, i, time.Now().Unix()))
		if err := v.s3Service.DownloadFileToPath("", segmentKey, localSegmentPath); err != nil {
			v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to download segment file %d: %v", i, err))
			return err
		}
		localSegmentPaths = append(localSegmentPaths, localSegmentPath)
		defer func(p string) { _ = os.Remove(p) }(localSegmentPath)
	}

	// 计算总文件大小
	var totalSize int64
	playlistInfo, _ := os.Stat(localPlaylistPath)
	if playlistInfo != nil {
		totalSize += playlistInfo.Size()
	}
	for _, segmentPath := range localSegmentPaths {
		segmentInfo, _ := os.Stat(segmentPath)
		if segmentInfo != nil {
			totalSize += segmentInfo.Size()
		}
	}

	// 创建统一的目录结构：所有文件（.m3u8 和 .ts）都在同一目录
	timestamp := time.Now().Unix()
	baseDir := fmt.Sprintf("transcoded/%d/%d_%d", file.ContentID, fileID, timestamp)

	// 上传 .m3u8 播放列表文件到最终S3位置
	finalPlaylistS3Key := fmt.Sprintf("%s/%s.m3u8", baseDir, outputName)

	// 需要修改 .m3u8 文件内容，更新分片路径为正确的 API 路径
	modifiedPlaylistPath, err := v.modifyPlaylistFile(localPlaylistPath, outputName, fileID)
	if err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to modify playlist file: %v", err))
		return err
	}
	defer func() { _ = os.Remove(modifiedPlaylistPath) }()

	// 上传 .m3u8 到最终 S3 位置（失败时上面所有 defers 会清理本地文件）
	playlistURL, err := v.s3Service.UploadFileFromPath("", finalPlaylistS3Key, modifiedPlaylistPath, "application/vnd.apple.mpegurl")
	if err != nil {
		v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to upload playlist file: %v", err))
		return err
	}

	// 上传所有 .ts 分片文件到最终 S3 位置（任一失败时删除已上传的最终 m3u8/ts）
	segmentS3Keys := make([]string, 0, len(localSegmentPaths))
	for i, localSegmentPath := range localSegmentPaths {
		originalSegmentName := fmt.Sprintf("%s_%03d.ts", outputName, i)
		finalSegmentS3Key := fmt.Sprintf("%s/%s", baseDir, originalSegmentName)
		_, err := v.s3Service.UploadFileFromPath("", finalSegmentS3Key, localSegmentPath, "video/mp2t")
		if err != nil {
			v.updateFileError(fileID, entity.StageFailed, fmt.Sprintf("Failed to upload segment file %d: %v", i, err))
			v.deleteS3Keys("", append([]string{finalPlaylistS3Key}, segmentS3Keys...)...)
			return err
		}
		segmentS3Keys = append(segmentS3Keys, finalSegmentS3Key)
	}

	// 删除临时S3文件
	v.s3Service.DeleteFile("", tempPlaylistS3Key)
	for _, tempSegmentS3Key := range tempSegmentS3Keys {
		if segmentKey, ok := tempSegmentS3Key.(string); ok {
			v.s3Service.DeleteFile("", segmentKey)
		}
	}

	// 更新文件记录（失败时删除已上传的最终 m3u8/ts）
	file.TranscodedFilePath = finalPlaylistS3Key
	file.TranscodedFileURL = playlistURL
	file.TranscodedFileSize = totalSize
	file.Stage = entity.StageTranscodedUploaded
	if err := v.fileRepo.Update(file); err != nil {
		v.deleteS3Keys("", append([]string{finalPlaylistS3Key}, segmentS3Keys...)...)
		return fmt.Errorf("failed to update file: %w", err)
	}

	// 发送原始文件删除任务消息（失败时删除已上传的最终 m3u8/ts）
	deleteMsg := &KafkaMessage{
		Type:      "original_delete",
		ContentID: file.ContentID,
		FileID:    fileID,
		Stage:     string(entity.StageTranscodedUploaded),
		Data: map[string]interface{}{
			"s3_key": file.OriginalFilePath,
		},
	}
	if err := v.kafkaService.SendMessage(TopicOriginalDelete, deleteMsg); err != nil {
		v.deleteS3Keys("", append([]string{finalPlaylistS3Key}, segmentS3Keys...)...)
		return fmt.Errorf("failed to send delete message: %w", err)
	}
	return nil
}

// handleOriginalDelete 处理原始文件删除任务
func (v *VideoProcessorConsumer) handleOriginalDelete(msg *KafkaMessage) error {
	fileID := msg.FileID

	// 获取文件记录
	file, err := v.fileRepo.GetByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 非预期阶段时 ack 消息，避免重复重试
	if !file.CanDeleteOriginal() {
		return nil
	}

	// 从S3删除原始文件
	if err := v.s3Service.DeleteFile("", file.OriginalFilePath); err != nil {
		// 记录错误但不失败（文件可能已经被删除）
		fmt.Printf("Failed to delete original file %s: %v\n", file.OriginalFilePath, err)
	}

	// 更新文件记录
	file.Stage = entity.StageCompleted
	now := time.Now()
	file.ProcessedAt = &now
	if err := v.fileRepo.Update(file); err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	// 检查所有文件是否都处理完成
	files, err := v.fileRepo.GetByContentID(file.ContentID)
	if err == nil {
		allCompleted := true
		for _, f := range files {
			if f.Stage != entity.StageCompleted && f.Stage != entity.StageFailed {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			// 更新内容状态为就绪
			content, err := v.contentRepo.GetByID(file.ContentID)
			if err == nil {
				content.Status = entity.ContentStatusReady
				now := time.Now()
				content.ProcessedAt = &now
				v.contentRepo.Update(content)
			}
		}
	}

	return nil
}

// modifyPlaylistFile 修改 .m3u8 文件内容，将分片路径改为 API 路径格式
func (v *VideoProcessorConsumer) modifyPlaylistFile(playlistPath, outputName string, fileID uint) (string, error) {
	// 读取原始 .m3u8 文件
	file, err := os.Open(playlistPath)
	if err != nil {
		return "", fmt.Errorf("failed to open playlist file: %w", err)
	}
	defer file.Close()

	// 创建修改后的文件
	modifiedPath := playlistPath + ".modified"
	modifiedFile, err := os.Create(modifiedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create modified playlist file: %w", err)
	}
	defer modifiedFile.Close()

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(modifiedFile)

	for scanner.Scan() {
		line := scanner.Text()

		// 如果是分片文件行（以 .ts 结尾且不是注释）
		if strings.HasSuffix(line, ".ts") && !strings.HasPrefix(line, "#") {
			// 提取分片文件名
			segmentName := filepath.Base(line)
			// 替换为 API 路径格式
			line = fmt.Sprintf("/api/content/files/%d/stream?segment=%s", fileID, segmentName)
		}

		writer.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		os.Remove(modifiedPath)
		return "", fmt.Errorf("failed to read playlist file: %w", err)
	}

	if err := writer.Flush(); err != nil {
		os.Remove(modifiedPath)
		return "", fmt.Errorf("failed to write modified playlist file: %w", err)
	}

	return modifiedPath, nil
}

// deleteS3Keys 删除多个 S3 对象（忽略单次删除失败，用于出错时清理）
func (v *VideoProcessorConsumer) deleteS3Keys(bucket string, keys ...string) {
	for _, key := range keys {
		_ = v.s3Service.DeleteFile(bucket, key)
	}
}

// updateFileError 更新文件错误状态
func (v *VideoProcessorConsumer) updateFileError(fileID uint, stage entity.FileProcessingStage, errorMsg string) {
	file, err := v.fileRepo.GetByID(fileID)
	if err != nil {
		fmt.Printf("Failed to get file for error update: %v\n", err)
		return
	}

	file.Stage = stage
	file.ProcessingError = errorMsg
	now := time.Now()
	file.ProcessedAt = &now
	v.fileRepo.Update(file)
}
