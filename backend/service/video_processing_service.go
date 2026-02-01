package service

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VideoProcessingService 视频处理服务接口
type VideoProcessingService interface {
	// GenerateGif 从视频生成GIF
	GenerateGif(videoPath, outputPath string, startTime float64, duration float64) error

	// GenerateGifSampled 从整段视频中均匀采样 numFrames 帧生成GIF
	GenerateGifSampled(videoPath, outputPath string, duration float64, numFrames int) error

	// TranscodeVideo 转码视频为HLS格式（.m3u8 + .ts 分片）
	TranscodeVideo(inputPath, outputDir, outputName string, options *TranscodeOptions) (*HLSOutput, error)

	// GetVideoInfo 获取视频信息
	GetVideoInfo(videoPath string) (*VideoInfo, error)
}

// HLSOutput HLS转码输出结果
type HLSOutput struct {
	PlaylistPath string   // .m3u8 播放列表文件路径
	SegmentPaths []string // .ts 分片文件路径列表
	OutputDir    string   // 输出目录
}

// TranscodeOptions 转码选项
type TranscodeOptions struct {
	Codec      string // 视频编码器 (h264, h265, vp9等)
	Bitrate    string // 比特率 (如: 2000k)
	Resolution string // 分辨率 (如: 1920x1080)
	Format     string // 输出格式 (mp4, webm等)
	Quality    string // 质量预设 (如: medium, high)
}

// VideoInfo 视频信息
type VideoInfo struct {
	Width    int     // 宽度
	Height   int     // 高度
	Duration float64 // 时长（秒）
	Bitrate  string  // 比特率
	Codec    string  // 编码器
	FPS      float64 // 帧率
}

// videoProcessingService 视频处理服务实现
type videoProcessingService struct {
	ffmpegPath  string
	ffprobePath string
	tempDir     string
}

// NewVideoProcessingService 创建视频处理服务实例
func NewVideoProcessingService() (VideoProcessingService, error) {
	// 从环境变量获取FFmpeg路径
	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 默认使用系统PATH中的ffmpeg
	}

	ffprobePath := os.Getenv("FFPROBE_PATH")
	if ffprobePath == "" {
		ffprobePath = "ffprobe" // 默认使用系统PATH中的ffprobe
	}

	// 检查FFmpeg是否可用
	if _, err := exec.LookPath(ffmpegPath); err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// 检查FFprobe是否可用
	if _, err := exec.LookPath(ffprobePath); err != nil {
		return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	// 创建临时目录
	tempDir := os.Getenv("TEMP_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	return &videoProcessingService{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		tempDir:     tempDir,
	}, nil
}

// toFFmpegPath 将路径转为 FFmpeg 可识别的形式（Windows 下反斜杠改为正斜杠，避免打开失败）
func toFFmpegPath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// GenerateGif 从视频生成GIF
func (v *videoProcessingService) GenerateGif(videoPath, outputPath string, startTime float64, duration float64) error {
	// 确保输出目录存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 构建FFmpeg命令
	// 使用palette生成更高质量的GIF
	palettePath := outputPath + ".png"

	// 第一步：生成调色板
	paletteCmd := exec.Command(v.ffmpegPath,
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", videoPath,
		"-vf", "fps=10,scale=320:-1:flags=lanczos,palettegen",
		"-y",
		palettePath,
	)

	if err := paletteCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate palette: %w", err)
	}
	defer os.Remove(palettePath) // 清理临时调色板文件

	// 第二步：使用调色板生成GIF
	gifCmd := exec.Command(v.ffmpegPath,
		"-ss", fmt.Sprintf("%.2f", startTime),
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", videoPath,
		"-i", palettePath,
		"-lavfi", "fps=10,scale=320:-1:flags=lanczos[x];[x][1:v]paletteuse",
		"-y",
		outputPath,
	)

	if err := gifCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate GIF: %w", err)
	}

	return nil
}

// GenerateGifSampled 从整段视频中均匀采样 numFrames 帧生成GIF（全片时长内等间隔取帧）
// 使用单条 FFmpeg filter_complex，不写调色板文件，避免 Windows 下调色板文件未创建问题
func (v *videoProcessingService) GenerateGifSampled(videoPath, outputPath string, duration float64, numFrames int) error {
	if duration <= 0 || numFrames <= 0 {
		return fmt.Errorf("duration and numFrames must be positive")
	}
	outDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	outputPath, _ = filepath.Abs(outputPath)
	videoPath, _ = filepath.Abs(videoPath)

	// 采样率：numFrames 帧 / duration 秒；fps 至少 1 避免 “No filtered frames”（极低 fps 时 filter 无输出）
	fpsVal := float64(numFrames) / duration
	if fpsVal < 1 {
		fpsVal = 1
	}
	fpsArg := fmt.Sprintf("%.4f", fpsVal)
	// 单命令：split -> 一路 palettegen，一路 scale -> paletteuse，直接输出 GIF
	filterComplex := fmt.Sprintf("split[s0][s1];[s0]fps=%s,scale=320:-1:flags=lanczos,palettegen=stats_mode=single[p];[s1]fps=%s,scale=320:-1:flags=lanczos[x];[x][p]paletteuse=new=1", fpsArg, fpsArg)

	cmd := exec.Command(v.ffmpegPath,
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", videoPath,
		"-filter_complex", filterComplex,
		"-y",
		outputPath,
	)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		msg := err.Error()
		if stderr.Len() > 0 {
			msg = strings.TrimSpace(stderr.String())
			if len(msg) > 1500 {
				msg = "... " + msg[len(msg)-1500:]
			}
		}
		return fmt.Errorf("failed to generate GIF: %w (ffmpeg: %s)", err, msg)
	}
	return nil
}

// TranscodeVideo 转码视频为HLS格式（.m3u8 + .ts 分片）
func (v *videoProcessingService) TranscodeVideo(inputPath, outputDir, outputName string, options *TranscodeOptions) (*HLSOutput, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// 默认转码选项（x264 preset 须为 ultrafast|superfast|veryfast|faster|fast|medium|slow|slower|veryslow|placebo，不能用 high）
	if options == nil {
		options = &TranscodeOptions{
			Codec:      "libx264",
			Bitrate:    "2000k",
			Resolution: "1920x1080", // 默认转码到 1080p
			Format:     "hls",
			Quality:    "medium",    // x264 preset：medium 平衡速度与质量
		}
	}

	// 如果没有指定分辨率，使用默认的 1920x1080
	if options.Resolution == "" {
		options.Resolution = "1920x1080"
	}

	// 构建输出路径
	playlistPath := filepath.Join(outputDir, outputName+".m3u8")
	segmentPattern := filepath.Join(outputDir, outputName+"_%03d.ts")

	// 构建FFmpeg命令用于HLS转码
	args := []string{
		"-i", inputPath,
		"-c:v", options.Codec,
		"-preset", options.Quality,
	}

	// 添加比特率
	if options.Bitrate != "" {
		args = append(args, "-b:v", options.Bitrate)
	}

	// 总是添加分辨率缩放（保持宽高比）；使用 -2 保证宽高为偶数（libx264 要求）
	resolutionParts := strings.Split(options.Resolution, "x")
	if len(resolutionParts) == 2 {
		args = append(args, "-vf", fmt.Sprintf("scale=%s:-2", resolutionParts[0]))
	} else {
		args = append(args, "-vf", "scale=1920:-2")
	}

	// 音频编码
	args = append(args, "-c:a", "aac", "-b:a", "128k")

	// HLS 特定参数
	args = append(args,
		"-hls_time", "10", // 每个分片10秒
		"-hls_list_size", "0", // 0表示包含所有分片
		"-hls_segment_filename", segmentPattern, // 分片文件命名模式
		"-hls_flags", "independent_segments", // 独立分片标志
		"-y",
		playlistPath, // 输出 .m3u8 播放列表
	)

	cmd := exec.Command(v.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 运行转码命令
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			ffmpegErr := strings.TrimSpace(stderr.String())
			// 取末尾一段（错误信息通常在 stderr 末尾，前面是 version banner）
			if len(ffmpegErr) > 1500 {
				ffmpegErr = "... " + ffmpegErr[len(ffmpegErr)-1500:]
			}
			return nil, fmt.Errorf("failed to transcode video to HLS: %w (ffmpeg: %s)", err, ffmpegErr)
		}
		return nil, fmt.Errorf("failed to transcode video to HLS: %w", err)
	}

	// 查找所有生成的 .ts 分片文件
	segmentPaths, err := v.findSegmentFiles(outputDir, outputName)
	if err != nil {
		return nil, fmt.Errorf("failed to find segment files: %w", err)
	}

	return &HLSOutput{
		PlaylistPath: playlistPath,
		SegmentPaths: segmentPaths,
		OutputDir:    outputDir,
	}, nil
}

// findSegmentFiles 查找所有生成的 .ts 分片文件
func (v *videoProcessingService) findSegmentFiles(outputDir, outputName string) ([]string, error) {
	files, err := ioutil.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	var segmentPaths []string
	prefix := outputName + "_"
	suffix := ".ts"

	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
				segmentPaths = append(segmentPaths, filepath.Join(outputDir, name))
			}
		}
	}

	// 按文件名排序（确保顺序正确）
	for i := 0; i < len(segmentPaths)-1; i++ {
		for j := i + 1; j < len(segmentPaths); j++ {
			if segmentPaths[i] > segmentPaths[j] {
				segmentPaths[i], segmentPaths[j] = segmentPaths[j], segmentPaths[i]
			}
		}
	}

	return segmentPaths, nil
}

// GetVideoInfo 获取视频信息
func (v *videoProcessingService) GetVideoInfo(videoPath string) (*VideoInfo, error) {
	// 使用ffprobe获取视频信息
	cmd := exec.Command(v.ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration,bit_rate,codec_name,r_frame_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 6 {
		return nil, fmt.Errorf("unexpected ffprobe output format")
	}

	info := &VideoInfo{}

	// 解析宽度和高度
	fmt.Sscanf(lines[0], "%d", &info.Width)
	fmt.Sscanf(lines[1], "%d", &info.Height)
	fmt.Sscanf(lines[2], "%f", &info.Duration)
	info.Bitrate = strings.TrimSpace(lines[3])
	info.Codec = strings.TrimSpace(lines[4])

	// 解析帧率 (格式: 30/1)
	fpsParts := strings.Split(strings.TrimSpace(lines[5]), "/")
	if len(fpsParts) == 2 {
		var num, den float64
		fmt.Sscanf(fpsParts[0], "%f", &num)
		fmt.Sscanf(fpsParts[1], "%f", &den)
		if den != 0 {
			info.FPS = num / den
		}
	}

	return info, nil
}
