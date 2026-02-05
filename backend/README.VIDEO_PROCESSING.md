# 视频上传和处理系统

## 概述

这个系统实现了批量视频上传和处理功能，支持：
- 批量上传最多10个视频文件
- 自动生成GIF预览
- 使用FFmpeg转码视频
- 通过Kafka队列异步处理
- 使用AWS S3存储文件

## 处理流程

1. **上传阶段**: 用户上传视频文件到S3
2. **GIF生成阶段**: 从视频生成GIF预览（前5秒）
3. **转码阶段**: 使用FFmpeg转码视频（H.264编码）
4. **上传转码文件**: 将转码后的文件上传回S3
5. **删除原始文件**: 删除S3中的原始视频文件

所有阶段都通过Kafka队列异步处理，支持并行处理多个文件。

## 环境变量配置

在 `.env` 文件中配置以下环境变量：

```env
# AWS S3 配置
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
AWS_S3_BUCKET=your-bucket-name

# Kafka 配置
KAFKA_BROKERS=localhost:9092

# FFmpeg 配置（可选，如果不在PATH中）
FFMPEG_PATH=/usr/local/bin/ffmpeg
FFPROBE_PATH=/usr/local/bin/ffprobe

# 使用 GPU 加速转码（可选，需 FFmpeg 带 NVENC/CUDA）
# 设为 1 时使用 NVIDIA h264_nvenc + scale_cuda，可显著加快 HLS 转码
FFMPEG_USE_GPU=0

# 临时文件目录（可选）
TEMP_DIR=/tmp/video-processing
```

## 数据库迁移

系统会自动创建以下表：
- `content_files` - 存储每个文件的处理状态

## API 端点

### 批量上传视频

```
POST /api/content/videos/upload
Content-Type: multipart/form-data

参数:
- name: 内容名称（必填）
- description: 内容描述（可选）
- files: 视频文件（最多10个，必填）

响应:
{
  "message": "Videos uploaded successfully",
  "files": [
    {
      "id": 1,
      "file_name": "video1.mp4",
      "file_index": 0,
      "original_file_url": "https://...",
      "stage": "uploaded",
      "created_at": "2026-01-28T10:00:00Z"
    }
  ]
}
```

### 获取内容详情

```
GET /api/content/{id}

响应:
{
  "content": {
    "id": 1,
    "name": "My Videos",
    "description": "...",
    "type": "video",
    "status": "processing",
    "created_at": "2026-01-28T10:00:00Z"
  },
  "files": [
    {
      "id": 1,
      "file_name": "video1.mp4",
      "file_index": 0,
      "original_file_url": "https://...",
      "gif_file_url": "https://...",
      "transcoded_file_url": "https://...",
      "stage": "completed",
      "width": 1920,
      "height": 1080,
      "duration": 120.5
    }
  ]
}
```

## 文件处理阶段

- `uploaded` - 已上传到S3
- `gif_generated` - GIF已生成
- `transcoded` - 已转码
- `transcoded_uploaded` - 转码文件已上传
- `original_deleted` - 原始文件已删除
- `completed` - 处理完成
- `failed` - 处理失败

## Kafka Topics

- `gif-generation` - GIF生成任务
- `video-transcode` - 视频转码任务
- `transcode-upload` - 转码文件上传任务
- `original-delete` - 原始文件删除任务

## 依赖要求

1. **FFmpeg**: 需要安装FFmpeg和FFprobe
   ```bash
   # macOS
   brew install ffmpeg
   
   # Ubuntu/Debian
   sudo apt-get install ffmpeg
   
   # 或从 https://ffmpeg.org/download.html 下载
   ```

2. **Kafka**: 需要运行Kafka服务器
   ```bash
   # 使用Docker运行Kafka
   docker run -d --name kafka -p 9092:9092 apache/kafka:latest
   ```

3. **AWS S3**: 需要配置AWS凭证和S3存储桶

## 安装依赖

```bash
cd backend
go mod tidy
```

## 运行

```bash
cd backend
go run main.go
```

系统会自动：
1. 初始化数据库连接
2. 运行数据库迁移
3. 初始化S3服务
4. 初始化Kafka服务
5. 启动Kafka消费者处理视频任务

## 注意事项

1. 确保FFmpeg已安装并在PATH中，或设置`FFMPEG_PATH`环境变量
2. 确保Kafka服务器正在运行
3. 确保AWS凭证配置正确且有S3访问权限
4. 临时目录需要有足够的磁盘空间存储下载和处理的视频文件
5. 处理大文件时可能需要较长时间，建议使用后台任务或消息队列

## 故障排除

### FFmpeg未找到
- 检查FFmpeg是否已安装：`ffmpeg -version`
- 设置`FFMPEG_PATH`环境变量指向FFmpeg可执行文件路径

### GPU 加速（可选）

HLS 转码可使用 **NVIDIA GPU（NVENC）** 加速，显著缩短转码时间：

1. **环境变量**：设置 `FFMPEG_USE_GPU=1`（或在调用 `TranscodeVideo` 时传入 `TranscodeOptions.UseGPU = true`）。
2. **FFmpeg 要求**：需使用带 NVENC/CUDA 的 FFmpeg 构建（`--enable-nvenc --enable-cuda`）。系统自带的 `apt install ffmpeg` / `apk add ffmpeg` 通常**不包含** NVENC，需自行编译或使用 [NVIDIA 官方容器](https://docs.nvidia.com/video-technologies/video-codec-sdk/ffmpeg-with-nvidia-gpu/)。
3. **管线**：启用后使用 `-hwaccel cuda` 解码、`scale_cuda` 缩放、`h264_nvenc` 编码；未启用或 GPU 不可用时使用 CPU（libx264）。
4. **Docker/K8s**：若在容器内使用 GPU，需挂载 NVIDIA 驱动并设置 `NVIDIA_VISIBLE_DEVICES` 等，参考 NVIDIA Container Toolkit。

### Kafka连接失败
- 检查Kafka服务器是否运行：`docker ps | grep kafka`
- 检查`KAFKA_BROKERS`环境变量是否正确

### Kafka 在 Windows 上因日志清理失败并退出（推荐用 Docker）
若在 **Windows 上原生运行 Kafka**（未用 Docker），可能遇到：
- 错误信息：`Error while deleting segments for analytics-topic-3 ... The process cannot access the file because it is being used by another process`
- 原因：Windows 下日志保留清理时重命名/删除 segment 文件会被文件锁阻止，导致 broker 将 log 目录标记为失败并退出。

**推荐做法：用 Docker 跑 Kafka（避免 Windows 文件锁）**
```bash
cd backend
docker-compose up -d zookeeper kafka
# 后端连接时使用 KAFKA_BROKERS=localhost:9092
```

**若必须在本机 Windows 跑 Kafka，恢复步骤：**
1. 停止 Kafka 进程（以及所有消费该集群的进程）。
2. 删除出问题的日志目录（例如 `C:\tmp\kafka-logs`，或仅删除 `C:\tmp\kafka-logs\analytics-topic-3`）。
3. 确认没有其他程序（杀毒、索引等）占用该目录后，重新启动 Kafka。
4. 如需减少此类问题，可适当增大 `log.retention.hours` 或 `log.retention.ms`，减少清理频率（仍可能再次发生）。

### S3上传失败
- 检查AWS凭证是否正确
- 检查S3存储桶是否存在且有写入权限
- 检查网络连接

### 视频处理失败
- 检查临时目录是否有足够的磁盘空间
- 检查视频文件格式是否支持
- 查看日志中的错误信息
