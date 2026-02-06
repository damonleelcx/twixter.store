package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	// multipart 单段最小约 5MB，这里用 10MB 以提升吞吐（参见 AWS multipart 文档）
	multipartPartSize  = 10 * 1024 * 1024
	multipartThreshold = 10 * 1024 * 1024 // 超过此大小走 multipart
)

// S3Service S3服务接口
type S3Service interface {
	// UploadFile 上传文件到S3
	UploadFile(bucket, key string, file io.Reader, contentType string) (string, error)

	// UploadFileFromPath 从本地路径上传文件到S3
	UploadFileFromPath(bucket, key, filePath string, contentType string) (string, error)

	// DownloadFile 从S3下载文件
	DownloadFile(bucket, key string) ([]byte, error)

	// DownloadFileToPath 从S3下载文件到本地路径
	DownloadFileToPath(bucket, key, localPath string) error

	// DeleteFile 从S3删除文件
	DeleteFile(bucket, key string) error

	// ListKeysByPrefix 按前缀列出对象 key（用于批量删除转码目录等）
	ListKeysByPrefix(bucket, prefix string) ([]string, error)

	// GetFileURL 获取文件的预签名URL
	GetFileURL(bucket, key string, expiresIn time.Duration) (string, error)

	// FileExists 检查文件是否存在
	FileExists(bucket, key string) (bool, error)

	// StreamFile 从S3流式传输文件，返回Reader和ContentType
	StreamFile(bucket, key string) (io.ReadCloser, string, error)

	// UploadFileFromPathWithProgress 从本地路径上传到 S3，使用 multipart 提升大文件速度，并回调进度（bytesRead, totalBytes）。
	// onProgress 可为 nil；大文件（>multipartPartSize）走 multipart upload。
	UploadFileFromPathWithProgress(bucket, key, filePath, contentType string, onProgress func(bytesRead, totalBytes int64)) (string, error)
}

// s3Service S3服务实现
type s3Service struct {
	s3Client   *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
	region     string
	endpoint   string // 自定义 endpoint（如另一家 S3 兼容存储），为空则用默认 AWS
}

// NewS3Service 创建S3服务实例
// 支持自定义 endpoint（如 S3 兼容的其他厂商）：设置 AWS_ENDPOINT 时会使用该 endpoint 并启用 path-style。
func NewS3Service() (S3Service, error) {
	// 从环境变量获取配置
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("AWS_S3_BUCKET")
	endpoint := strings.TrimSpace(os.Getenv("AWS_ENDPOINT"))

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS credentials not configured")
	}
	if region == "" {
		region = "us-east-1"
	}
	if bucket == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET not configured")
	}

	cfg := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	}
	if endpoint != "" {
		cfg.Endpoint = aws.String(endpoint)
		cfg.S3ForcePathStyle = aws.Bool(true)
	}

	// 创建AWS会话
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// 创建S3客户端
	s3Client := s3.New(sess)
	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)

	return &s3Service{
		s3Client:   s3Client,
		uploader:   uploader,
		downloader: downloader,
		bucket:     bucket,
		region:     region,
		endpoint:   endpoint,
	}, nil
}

// UploadFile 上传文件到S3
func (s *s3Service) UploadFile(bucket, key string, file io.Reader, contentType string) (string, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	// 读取文件内容
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 上传到S3
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// 返回文件URL：自定义 endpoint 时用 path-style，否则用虚拟主机式。
	// 必须带 https 协议，否则前端会把 URL 当相对路径请求到当前站点导致 404。
	if s.endpoint != "" {
		base := strings.TrimSuffix(s.endpoint, "/")
		if base != "" && !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
			base = "https://" + base
		}
		url := fmt.Sprintf("%s/%s/%s", base, bucket, key)
		return url, nil
	}
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, s.region, key)
	return url, nil
}

// removeLocalWithRetry 关闭后删除本地文件；Windows 上句柄释放可能延迟，失败时重试
func removeLocalWithRetry(filePath string) {
	const (
		retries  = 50
		interval = 200 * time.Millisecond
		winDelay = 200 * time.Millisecond // 首次删除前等待，给 Close() 后句柄释放时间
	)
	if runtime.GOOS == "windows" {
		time.Sleep(winDelay)
	}
	for i := 0; i < retries; i++ {
		err := os.Remove(filePath)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			return
		}
		if i < retries-1 && (strings.Contains(err.Error(), "being used") || strings.Contains(err.Error(), "used by another process")) {
			time.Sleep(interval)
			continue
		}
		LogError("Warning: failed to remove local file after S3 upload: %s: %v", filePath, err)
		return
	}
}

// UploadFileFromPath 从本地路径上传文件到S3；成功返回前会删除本地文件以释放磁盘空间
func (s *s3Service) UploadFileFromPath(bucket, key, filePath string, contentType string) (string, error) {
	url, err := s.UploadFileFromPathWithProgress(bucket, key, filePath, contentType, nil)
	removeLocalWithRetry(filePath)
	if err != nil {
		return "", err
	}
	return url, nil
}

// progressReader 包装 io.Reader，在读时回调进度（bytesRead, total）
type progressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	onProgress func(bytesRead, totalBytes int64)
}

func (p *progressReader) Read(b []byte) (n int, err error) {
	n, err = p.reader.Read(b)
	if n > 0 && p.onProgress != nil {
		p.read += int64(n)
		p.onProgress(p.read, p.total)
	}
	return n, err
}

// UploadFileFromPathWithProgress 从本地路径上传到 S3，支持进度回调和 multipart 大文件上传。
func (s *s3Service) UploadFileFromPathWithProgress(bucket, key, filePath, contentType string, onProgress func(bytesRead, totalBytes int64)) (string, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	total := stat.Size()
	if total == 0 {
		_, err = s.uploader.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        f,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload file to S3: %w", err)
		}
		return s.buildURL(bucket, key), nil
	}
	if total <= multipartThreshold {
		pr := &progressReader{reader: f, total: total, onProgress: onProgress}
		_, err = s.uploader.Upload(&s3manager.UploadInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        pr,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload file to S3: %w", err)
		}
		return s.buildURL(bucket, key), nil
	}
	// 大文件：multipart upload（CreateMultipartUpload -> UploadPart -> CompleteMultipartUpload）
	createOut, err := s.s3Client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create multipart upload: %w", err)
	}
	uploadID := createOut.UploadId
	defer func() {
		if uploadID != nil {
			_, _ = s.s3Client.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
				Bucket:   aws.String(bucket),
				Key:      aws.String(key),
				UploadId: uploadID,
			})
		}
	}()
	var parts []*s3.CompletedPart
	var partNumber int64 = 1
	var readSoFar int64
	buf := make([]byte, multipartPartSize)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			body := bytes.NewReader(buf[:n])
			upOut, upErr := s.s3Client.UploadPart(&s3.UploadPartInput{
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				UploadId:   uploadID,
				PartNumber: aws.Int64(partNumber),
				Body:       body,
			})
			if upErr != nil {
				return "", fmt.Errorf("failed to upload part %d: %w", partNumber, upErr)
			}
			readSoFar += int64(n)
			if onProgress != nil {
				onProgress(readSoFar, total)
			}
			parts = append(parts, &s3.CompletedPart{ETag: upOut.ETag, PartNumber: aws.Int64(partNumber)})
			partNumber++
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("failed to read file: %w", readErr)
		}
	}
	_, err = s.s3Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(bucket),
		Key:             aws.String(key),
		UploadId:        uploadID,
		MultipartUpload: &s3.CompletedMultipartUpload{Parts: parts},
	})
	if err != nil {
		return "", fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	uploadID = nil
	return s.buildURL(bucket, key), nil
}

func (s *s3Service) buildURL(bucket, key string) string {
	if s.endpoint != "" {
		base := strings.TrimSuffix(s.endpoint, "/")
		if base != "" && !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
			base = "https://" + base
		}
		return fmt.Sprintf("%s/%s/%s", base, bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, s.region, key)
}

// DownloadFile 从S3下载文件
func (s *s3Service) DownloadFile(bucket, key string) ([]byte, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := s.downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}

	return buf.Bytes(), nil
}

// DownloadFileToPath 从S3下载文件到本地路径
func (s *s3Service) DownloadFileToPath(bucket, key, localPath string) error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if bucket == "" {
		bucket = s.bucket
	}

	_, err = s.downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download file from S3: %w", err)
	}

	return nil
}

// DeleteFile 从S3删除文件
func (s *s3Service) DeleteFile(bucket, key string) error {
	if bucket == "" {
		bucket = s.bucket
	}

	_, err := s.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// ListKeysByPrefix 按前缀列出对象 key（用于批量删除转码目录等）
func (s *s3Service) ListKeysByPrefix(bucket, prefix string) ([]string, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	var keys []string
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	for {
		out, err := s.s3Client.ListObjectsV2(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects by prefix: %w", err)
		}
		for _, obj := range out.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
		if !aws.BoolValue(out.IsTruncated) {
			break
		}
		input.ContinuationToken = out.NextContinuationToken
	}
	return keys, nil
}

// GetFileURL 获取文件的预签名URL
func (s *s3Service) GetFileURL(bucket, key string, expiresIn time.Duration) (string, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	req, _ := s.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	url, err := req.Presign(expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url, nil
}

// FileExists 检查文件是否存在
func (s *s3Service) FileExists(bucket, key string) (bool, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	_, err := s.s3Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// 检查是否是404错误
		if _, ok := err.(interface{ StatusCode() int }); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// StreamFile 从S3流式传输文件，返回Reader和ContentType
func (s *s3Service) StreamFile(bucket, key string) (io.ReadCloser, string, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	result, err := s.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object from S3: %w", err)
	}

	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return result.Body, contentType, nil
}
