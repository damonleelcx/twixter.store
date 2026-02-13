package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitMongo 初始化 MongoDB 连接（用于聊天消息历史）；MONGO_URI 未设置时返回 nil
func InitMongo() (*mongo.Client, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		log.Println("MONGO_URI not set; chat message history will be disabled.")
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	log.Println("Successfully connected to MongoDB (chat store)")
	return client, nil
}
