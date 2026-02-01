package service

import (
	"errors"
	"io"
	"time"
)

// errServiceUnavailable 当 S3/Kafka 未配置时，上传、流式传输等操作返回此错误
var errServiceUnavailable = errors.New("service not configured (S3/Kafka unavailable)")

// noopS3Service 占位 S3 实现：所有方法返回 errServiceUnavailable，用于在未配置 S3 时仍能注册内容只读路由
type noopS3Service struct{}

// NoopS3Service 返回占位 S3 服务（config 包用）
func NoopS3Service() S3Service { return noopS3Service{} }

func (n noopS3Service) UploadFile(bucket, key string, file io.Reader, contentType string) (string, error) {
	return "", errServiceUnavailable
}
func (n noopS3Service) UploadFileFromPath(bucket, key, filePath string, contentType string) (string, error) {
	return "", errServiceUnavailable
}
func (n noopS3Service) DownloadFile(bucket, key string) ([]byte, error) {
	return nil, errServiceUnavailable
}
func (n noopS3Service) DownloadFileToPath(bucket, key, localPath string) error {
	return errServiceUnavailable
}
func (n noopS3Service) DeleteFile(bucket, key string) error {
	return errServiceUnavailable
}
func (n noopS3Service) GetFileURL(bucket, key string, expiresIn time.Duration) (string, error) {
	return "", errServiceUnavailable
}
func (n noopS3Service) FileExists(bucket, key string) (bool, error) {
	return false, errServiceUnavailable
}
func (n noopS3Service) StreamFile(bucket, key string) (io.ReadCloser, string, error) {
	return nil, "", errServiceUnavailable
}

// noopKafkaService 占位 Kafka 实现：所有方法返回 errServiceUnavailable
type noopKafkaService struct{}

// NoopKafkaService 返回占位 Kafka 服务（config 包用）
func NoopKafkaService() KafkaService { return noopKafkaService{} }

func (n noopKafkaService) SendMessage(topic string, message *KafkaMessage) error {
	return errServiceUnavailable
}
func (n noopKafkaService) StartConsumer(topic string, handler func(*KafkaMessage) error) error {
	return errServiceUnavailable
}
func (n noopKafkaService) Close() error {
	return nil
}
