package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	chatDBName         = "chat"
	sessionsCollName   = "sessions"
	sessionTTLHours    = 24 * 7 // 7 天短期会话过期
	defaultExpireAfter = 0      // expireAfterSeconds: 0 表示按 expires_at 字段值过期
)

// ChatMessage 单条消息（MongoDB 内嵌）
type ChatMessage struct {
	Role      string    `bson:"role" json:"role"`           // "user" | "assistant" | "system"
	Content   string    `bson:"content" json:"content"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// ChatSession 短期会话（MongoDB），用于消息历史 + TTL
type ChatSession struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SessionID string             `bson:"session_id" json:"session_id"` // 业务侧会话 ID（如 UUID）
	UserID    uint               `bson:"user_id" json:"user_id"`
	Messages  []ChatMessage      `bson:"messages" json:"messages"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// MongoChatStore MongoDB 聊天会话存储
type MongoChatStore struct {
	client *mongo.Client
	coll   *mongo.Collection
}

// NewMongoChatStore 创建 MongoDB 聊天存储并确保 TTL 索引
func NewMongoChatStore(client *mongo.Client) (*MongoChatStore, error) {
	coll := client.Database(chatDBName).Collection(sessionsCollName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TTL 索引：expires_at 到期后自动删除
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(defaultExpireAfter),
	}
	if _, err := coll.Indexes().CreateOne(ctx, idx); err != nil {
		return nil, err
	}

	return &MongoChatStore{client: client, coll: coll}, nil
}

// GetOrCreateSession 获取或创建会话；若不存在则创建并设置 expires_at
func (s *MongoChatStore) GetOrCreateSession(ctx context.Context, sessionID string, userID uint) (*ChatSession, error) {
	filter := bson.M{"session_id": sessionID}
	expiresAt := time.Now().Add(sessionTTLHours * time.Hour)
	now := time.Now()

	var doc ChatSession
	err := s.coll.FindOne(ctx, filter).Decode(&doc)
	if err == nil {
		// 延长过期时间
		_, _ = s.coll.UpdateOne(ctx, filter, bson.M{
			"$set": bson.M{"expires_at": expiresAt, "updated_at": now},
		})
		doc.ExpiresAt = expiresAt
		doc.UpdatedAt = now
		return &doc, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	doc = ChatSession{
		SessionID: sessionID,
		UserID:    userID,
		Messages:  []ChatMessage{},
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	res, err := s.coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		doc.ID = oid
	}
	doc.ExpiresAt = expiresAt
	return &doc, nil
}

// AppendMessage 追加一条消息并刷新 updated_at、expires_at
func (s *MongoChatStore) AppendMessage(ctx context.Context, sessionID string, role, content string) error {
	now := time.Now()
	expiresAt := now.Add(sessionTTLHours * time.Hour)
	msg := ChatMessage{Role: role, Content: content, CreatedAt: now}
	_, err := s.coll.UpdateOne(ctx, bson.M{"session_id": sessionID}, bson.M{
		"$push": bson.M{"messages": msg},
		"$set":  bson.M{"updated_at": now, "expires_at": expiresAt},
	})
	return err
}

// GetSession 仅获取会话（含消息历史）
func (s *MongoChatStore) GetSession(ctx context.Context, sessionID string) (*ChatSession, error) {
	var doc ChatSession
	err := s.coll.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}
