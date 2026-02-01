# 批量视频上传工具 (bulk_upload)

从本地文件夹读取视频文件，按与 API 相同的 S3 路径格式上传到 AWS S3，并走相同的 Kafka 流水线完成：GIF 生成、HLS 转码、转码上传、原文件删除；数据库记录完整，缩略图（GIF）会由流水线自动写入 `content.thumbnail_url`。

## 行为说明

- **上传者**：固定为 admin 用户（user_id: 7，email: admin@twixter.store，username: twixter_user）
- **内容**：name 与 description 由文件名解析；category 固定为 `dark`；price 固定为 10 credits
- **标签**：从文件名解析若干标签（如 `summer_beach_fun.mp4` → 标签 summer, beach, fun）
- **S3 路径**：与 `content_service.UploadVideos` 一致，原始视频为 `videos/{user_id}/{timestamp}_{filename}`
- **Kafka**：每条视频上传后发送到 `gif-generation`，由**本进程内启动的** `VideoProcessorConsumer` 依次消费并完成 GIF → 转码 → 上传转码 → 删除原文件，并最终将内容状态设为 ready、将首文件 GIF 写入 `content.thumbnail_url`。**无需另开 backend 服务**，本工具会同时消费 Kafka 并跑完整流水线。

## 环境与依赖

需在项目根或 backend 目录下配置 `.env`（或环境变量）：

- **数据库**：`DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`, `DB_SSLMODE`
- **AWS S3**：`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `AWS_S3_BUCKET`
- **Kafka**：`KAFKA_BROKERS`（如 `localhost:9092`）
- **FFmpeg / FFprobe**：本进程会启动 Kafka 消费者并执行 GIF 生成与 HLS 转码，需系统 PATH 中有 `ffmpeg` 与 `ffprobe`（或通过 `FFMPEG_PATH` / `FFPROBE_PATH` 指定）。若未安装，工具仍会上传并入队，但需由其他进程（如 backend 服务）消费 Kafka 完成处理。

## 使用方式

在仓库根目录或 backend 目录下执行：

```bash
# 指定视频所在文件夹（只处理 .mp4 .mov .webm .avi）
go run backend/cmd/bulk_upload/main.go -folder=./videos
```

或先编译再运行：

```bash
cd backend
go build -o bulk_upload ./cmd/bulk_upload/
./bulk_upload -folder=/path/to/videos
```

未指定 `-folder` 会报错并退出。**文件夹必须已存在**：若使用 `./videos`，请先创建并放入视频：

```bash
mkdir videos
# 将 .mp4 / .mov / .webm / .avi 放入 videos 目录后再运行
go run backend/cmd/bulk_upload/main.go -folder=./videos
```

也可用绝对路径指定已有视频目录。**在 Git Bash / WSL 等 Bash 下，请用正斜杠**，否则 `\` 会被转义导致路径错误：

- 推荐：`-folder=C:/Users/damon/Downloads/videos`
- 或在 CMD 下加引号：`-folder="C:\Users\damon\Downloads\videos"`
