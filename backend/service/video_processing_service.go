package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

// VideoProcessingService 视频处理服务接口
type VideoProcessingService interface {
	// GenerateGif 从视频生成GIF
	GenerateGif(videoPath, outputPath string, startTime float64, duration float64) error

	// GenerateGifSampled 从整段视频中均匀采样 numFrames 帧生成GIF
	GenerateGifSampled(videoPath, outputPath string, duration float64, numFrames int) error

	// BlurGif 从本地 GIF 文件生成模糊版并写入 outputPath（上传阶段用）
	BlurGif(inputPath, outputPath string) error

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
	Codec      string // 视频编码器 (libx264, h264_nvenc 等)
	Bitrate    string // 比特率 (如: 2000k)
	Resolution string // 分辨率 (如: 1920x1080)
	Format     string // 输出格式 (mp4, webm等)
	Quality    string // 质量预设：CPU 为 x264 preset(medium等)，NVENC 为 p1-p7(p4 平衡)
	UseGPU     bool   // 为 true 时使用 GPU 加速（NVIDIA NVENC），需 FFmpeg 带 --enable-nvenc
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

	// Windows 下传给 FFmpeg 的路径用正斜杠，避免反斜杠被转义导致输出到错误位置
	cmd := exec.Command(v.ffmpegPath,
		"-t", fmt.Sprintf("%.2f", duration),
		"-i", toFFmpegPath(videoPath),
		"-filter_complex", filterComplex,
		"-y",
		toFFmpegPath(outputPath),
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

const gifBlurRadius = 3
const maxBlurGifFrames = 12

func ensureDelayLen(d []int, n int) []int {
	if len(d) >= n {
		return d[:n]
	}
	out := make([]int, n)
	copy(out, d)
	last := 0
	if len(d) > 0 {
		last = d[len(d)-1]
	}
	for i := len(d); i < n; i++ {
		out[i] = last
	}
	return out
}

func ensureDisposalLen(d []byte, n int) []byte {
	if len(d) >= n {
		return d[:n]
	}
	out := make([]byte, n)
	copy(out, d)
	var last byte
	if len(d) > 0 {
		last = d[len(d)-1]
	}
	for i := len(d); i < n; i++ {
		out[i] = last
	}
	return out
}

// BlurGif 从本地 GIF 文件生成模糊版并写入 outputPath
func (v *videoProcessingService) BlurGif(inputPath, outputPath string) error {
	// 统一为绝对路径，避免 Windows 下工作目录/相对路径导致找不到文件
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		absInput = inputPath
	}
	// Windows 下 FFmpeg 刚写完文件时可能尚未可见，短暂重试
	const readRetries = 5
	const readRetryDelay = 300 * time.Millisecond
	var data []byte
	for i := 0; i < readRetries; i++ {
		data, err = os.ReadFile(absInput)
		if err == nil {
			break
		}
		errStr := err.Error()
		retryable := errors.Is(err, os.ErrNotExist) ||
			strings.Contains(errStr, "cannot find the file") ||
			strings.Contains(errStr, "no such file") ||
			strings.Contains(errStr, "The system cannot find the file specified")
		if i < readRetries-1 && retryable {
			time.Sleep(readRetryDelay)
			continue
		}
		return fmt.Errorf("read gif: %w", err)
	}
	img, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode gif: %w", err)
	}
	if len(img.Image) == 0 {
		return fmt.Errorf("empty gif")
	}
	n := len(img.Image)
	if n > maxBlurGifFrames {
		n = maxBlurGifFrames
	}
	blurredFrames := make([]*image.Paletted, n)
	workers := runtime.NumCPU()
	if workers > n {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	ch := make(chan int, n)
	for i := 0; i < n; i++ {
		ch <- i
	}
	close(ch)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				frame := img.Image[i]
				blurred := imaging.Blur(imaging.Clone(frame), gifBlurRadius)
				dst := image.NewPaletted(blurred.Bounds(), frame.Palette)
				draw.Draw(dst, dst.Bounds(), blurred, image.Point{}, draw.Src)
				blurredFrames[i] = dst
			}
		}()
	}
	wg.Wait()
	out := &gif.GIF{
		Image:           blurredFrames,
		Delay:           ensureDelayLen(img.Delay, len(blurredFrames)),
		Disposal:        ensureDisposalLen(img.Disposal, len(blurredFrames)),
		LoopCount:       img.LoopCount,
		Config:          img.Config,
		BackgroundIndex: img.BackgroundIndex,
	}
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()
	if err := gif.EncodeAll(outFile, out); err != nil {
		return fmt.Errorf("encode gif: %w", err)
	}
	return nil
}

// TranscodeVideo 转码视频为HLS格式（.m3u8 + .ts 分片）
func (v *videoProcessingService) TranscodeVideo(inputPath, outputDir, outputName string, options *TranscodeOptions) (*HLSOutput, error) {
	// 规范为绝对路径，避免 Windows 下相对路径或不同写法导致目录“找不到”
	absOutputDir, err := filepath.Abs(filepath.Clean(outputDir))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output directory: %w", err)
	}
	outputDir = absOutputDir
	// 确保输出目录存在（含父目录）
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// 默认转码选项（x264 preset 须为 ultrafast|superfast|veryfast|faster|fast|medium|slow|slower|veryslow|placebo，不能用 high）
	if options == nil {
		options = &TranscodeOptions{
			Codec:      "libx264",
			Bitrate:    "2000k",
			Resolution: "1920x1080", // 默认转码到 1080p
			Format:     "hls",
			Quality:    "medium", // x264 preset：medium 平衡速度与质量
		}
	}

	// 如果没有指定分辨率，使用默认的 1920x1080
	if options.Resolution == "" {
		options.Resolution = "1920x1080"
	}

	// 是否使用 GPU：显式设置或环境变量 FFMPEG_USE_GPU=1
	useGPU := options.UseGPU
	if !useGPU && os.Getenv("FFMPEG_USE_GPU") == "1" {
		useGPU = true
	}

	// 构建输出路径
	playlistPath := filepath.Join(outputDir, outputName+".m3u8")
	segmentPattern := filepath.Join(outputDir, outputName+"_%03d.ts")

	var args []string
	if useGPU {
		// GPU 管线：CUDA 解码 + scale_cuda + h264_nvenc（需 FFmpeg 编译时启用 nvenc/cuda）
		codec := options.Codec
		if codec == "" || codec == "libx264" {
			codec = "h264_nvenc"
		}
		quality := options.Quality
		if quality == "" || quality == "medium" {
			quality = "p4" // NVENC p1(最快)..p7(最慢/质量最好)
		}
		args = []string{
			"-hwaccel", "cuda", "-hwaccel_output_format", "cuda",
			"-i", toFFmpegPath(inputPath),
			"-c:v", codec,
			"-preset", quality,
		}
		if options.Bitrate != "" {
			args = append(args, "-b:v", options.Bitrate)
		}
		resolutionParts := strings.Split(options.Resolution, "x")
		scaleArg := "1920:-2"
		if len(resolutionParts) == 2 {
			scaleArg = resolutionParts[0] + ":-2"
		}
		// scale_cuda 保持宽高比；-2 保证偶数
		args = append(args, "-vf", "scale_cuda="+scaleArg)
	} else {
		// 原有 CPU 管线：libx264
		codec := options.Codec
		if codec == "" {
			codec = "libx264"
		}
		quality := options.Quality
		if quality == "" {
			quality = "medium"
		}
		args = []string{
			"-i", toFFmpegPath(inputPath),
			"-c:v", codec,
			"-preset", quality,
		}
		if options.Bitrate != "" {
			args = append(args, "-b:v", options.Bitrate)
		}
		resolutionParts := strings.Split(options.Resolution, "x")
		if len(resolutionParts) == 2 {
			args = append(args, "-vf", fmt.Sprintf("scale=%s:-2", resolutionParts[0]))
		} else {
			args = append(args, "-vf", "scale=1920:-2")
		}
	}

	// 音频编码
	args = append(args, "-c:a", "aac", "-b:a", "128k")

	// HLS 特定参数（路径统一转成 FFmpeg 可识别的形式，避免 Windows 反斜杠问题）
	args = append(args,
		"-hls_time", "10", // 每个分片10秒
		"-hls_list_size", "0", // 0表示包含所有分片
		"-hls_segment_filename", toFFmpegPath(segmentPattern), // 分片文件命名模式
		"-hls_flags", "independent_segments", // 独立分片标志
		"-y",
		toFFmpegPath(playlistPath), // 输出 .m3u8 播放列表
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

	// 转码后确认输出目录仍存在（Windows 下路径不一致或目录未创建时便于排查）
	if fi, err := os.Stat(outputDir); err != nil {
		return nil, fmt.Errorf("output directory missing after ffmpeg (path: %s): %w", outputDir, err)
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("output path is not a directory: %s", outputDir)
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
	dir := filepath.Clean(outputDir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read output dir %s: %w", dir, err)
	}

	var segmentPaths []string
	prefix := outputName + "_"
	suffix := ".ts"

	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
				segmentPaths = append(segmentPaths, filepath.Join(dir, name))
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
