package service

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"
)

// KafkaMessage Kafka消息结构
type KafkaMessage struct {
	Type      string                 `json:"type"`       // 消息类型
	ContentID uint                   `json:"content_id"` // 内容ID
	FileID    uint                   `json:"file_id"`    // 文件ID
	Stage     string                 `json:"stage"`      // 处理阶段
	Data      map[string]interface{} `json:"data"`       // 附加数据
	Timestamp time.Time              `json:"timestamp"`  // 时间戳
}

// KafkaService Kafka服务接口
type KafkaService interface {
	// SendMessage 发送消息到Kafka
	SendMessage(topic string, message *KafkaMessage) error

	// StartConsumer 启动消费者
	StartConsumer(topic string, handler func(*KafkaMessage) error) error

	// Close 关闭Kafka连接
	Close() error
}

// kafkaService Kafka服务实现
type kafkaService struct {
	producer sarama.SyncProducer
	consumer sarama.Consumer
	brokers  []string
}

// NewKafkaService 创建Kafka服务实例
func NewKafkaService() (KafkaService, error) {
	// 从环境变量获取Kafka brokers
	// 注意：
	// - 在 Docker Compose 中：docker-compose.yml 会自动设置 KAFKA_BROKERS=kafka:9092（如果 .env 中没有）
	// - 本地开发时：需要在 .env 中设置 KAFKA_BROKERS=localhost:9092
	// - 这个默认值仅用于本地开发，在 Docker 中不会被使用（因为 docker-compose.yml 会设置环境变量）
	brokerList := os.Getenv("KAFKA_BROKERS")
	if brokerList == "" {
		brokerList = "localhost:9092" // 仅用于本地开发，Docker 中应通过环境变量设置
	}

	brokers := []string{brokerList}

	// 创建生产者配置
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Retry.Max = 5

	// 创建生产者
	producer, err := sarama.NewSyncProducer(brokers, producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// 创建消费者配置
	consumerConfig := sarama.NewConfig()
	consumerConfig.Consumer.Return.Errors = true
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	// 创建消费者
	consumer, err := sarama.NewConsumer(brokers, consumerConfig)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &kafkaService{
		producer: producer,
		consumer: consumer,
		brokers:  brokers,
	}, nil
}

// SendMessage 发送消息到Kafka
func (k *kafkaService) SendMessage(topic string, message *KafkaMessage) error {
	// 设置时间戳
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// 序列化消息
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 创建Kafka消息
	kafkaMessage := &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(messageBytes),
		Timestamp: message.Timestamp,
	}

	// 发送消息
	partition, offset, err := k.producer.SendMessage(kafkaMessage)
	if err != nil {
		return fmt.Errorf("failed to send message to Kafka: %w", err)
	}

	fmt.Printf("Message sent to topic %s, partition %d, offset %d\n", topic, partition, offset)
	return nil
}

// StartConsumer 启动消费者
func (k *kafkaService) StartConsumer(topic string, handler func(*KafkaMessage) error) error {
	// 获取主题的所有分区
	partitions, err := k.consumer.Partitions(topic)
	if err != nil {
		return fmt.Errorf("failed to get partitions for topic %s: %w", topic, err)
	}

	// 为每个分区创建消费者
	for _, partition := range partitions {
		partitionConsumer, err := k.consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
		if err != nil {
			return fmt.Errorf("failed to create partition consumer: %w", err)
		}

		// 启动goroutine处理消息
		go func(pc sarama.PartitionConsumer) {
			defer pc.AsyncClose()

			for {
				select {
				case message := <-pc.Messages():
					var kafkaMsg KafkaMessage
					if err := json.Unmarshal(message.Value, &kafkaMsg); err != nil {
						fmt.Printf("Failed to unmarshal message: %v\n", err)
						continue
					}

					// 处理消息
					if err := handler(&kafkaMsg); err != nil {
						fmt.Printf("Failed to handle message: %v\n", err)
						// 可以在这里实现重试逻辑
					}

				case err := <-pc.Errors():
					if err != nil {
						fmt.Printf("Kafka consumer error: %v\n", err)
					}
				}
			}
		}(partitionConsumer)
	}

	return nil
}

// Close 关闭Kafka连接
func (k *kafkaService) Close() error {
	var errs []error

	if err := k.producer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close producer: %w", err))
	}

	if err := k.consumer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close consumer: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing Kafka service: %v", errs)
	}

	return nil
}

// Kafka Topics
const (
	TopicVideoUpload     = "video-upload"     // 视频上传完成
	TopicGifGeneration   = "gif-generation"   // GIF生成任务
	TopicVideoTranscode  = "video-transcode"  // 视频转码任务
	TopicTranscodeUpload = "transcode-upload" // 转码文件上传任务
	TopicOriginalDelete  = "original-delete"  // 原始文件删除任务
)
