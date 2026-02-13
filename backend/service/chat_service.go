package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"backend/entity"
	"backend/repository"
	"backend/store"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ErrInsufficientCredits 积分不足（每条消息 1 积分）
var ErrInsufficientCredits = errors.New("insufficient credits: 1 credit per message")

// ChatService 聊天：人格存 Postgres，消息历史存 MongoDB，每条消息扣 1 积分
type ChatService struct {
	personaRepo repository.ChatPersonaRepository
	walletRepo  repository.WalletRepository
	mongoChat   *store.MongoChatStore
	llmClient   *LLMClient
}

// NewChatService 创建聊天服务（mongoChat / llmClient 可为 nil，对应能力不可用）
func NewChatService(
	personaRepo repository.ChatPersonaRepository,
	walletRepo repository.WalletRepository,
	mongoChat *store.MongoChatStore,
	llmClient *LLMClient,
) *ChatService {
	return &ChatService{personaRepo: personaRepo, walletRepo: walletRepo, mongoChat: mongoChat, llmClient: llmClient}
}

// NeedOnboarding 若用户尚未填写名字与人格则返回 true
func (s *ChatService) NeedOnboarding(userID uint) (bool, error) {
	_, err := s.personaRepo.GetByUserID(userID)
	if err == gorm.ErrRecordNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

// GetPersona 获取用户人格；不存在返回 nil, nil
func (s *ChatService) GetPersona(userID uint) (*entity.ChatPersona, error) {
	return s.personaRepo.GetByUserID(userID)
}

// SavePersona 首次聊天时保存名字与人格到 Postgres
func (s *ChatService) SavePersona(userID uint, name string, personality entity.PersonalitySchema) error {
	jb, err := json.Marshal(personality)
	if err != nil {
		return err
	}
	p := &entity.ChatPersona{
		UserID:     userID,
		Name:       name,
		Personality: datatypes.JSON(jb),
	}
	return s.personaRepo.Upsert(p)
}

// GetOrCreateSession 获取或创建会话 ID；若无 Mongo 则返回仅内存的 sessionID
func (s *ChatService) GetOrCreateSession(ctx context.Context, sessionID string, userID uint) (string, *store.ChatSession, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	if s.mongoChat == nil {
		return sessionID, &store.ChatSession{SessionID: sessionID, UserID: userID, Messages: nil}, nil
	}
	sess, err := s.mongoChat.GetOrCreateSession(ctx, sessionID, userID)
	if err != nil {
		return "", nil, err
	}
	return sess.SessionID, sess, nil
}

// BuildPrompt 根据人格与消息历史构建发给 LLM 的 prompt
func (s *ChatService) BuildPrompt(persona *entity.ChatPersona, messages []store.ChatMessage) string {
	var sb strings.Builder
	if persona != nil && len(persona.Personality) > 0 {
		sb.WriteString("You are chatting with a user who has set the following persona. Stay in character.\n")
		sb.WriteString("Name: " + persona.Name + "\n")
		sb.WriteString("Personality (JSON): " + string(persona.Personality) + "\n\n")
	}
	sb.WriteString("Rules: Do not suggest the user chat elsewhere or include links to external chat pages. Never say things like \"If you want to chat with [name], you can do so here\" or include share links.\n\n")
	sb.WriteString("Conversation:\n")
	for _, m := range messages {
		sb.WriteString(m.Role + ": " + m.Content + "\n")
	}
	sb.WriteString("assistant: ")
	return sb.String()
}

// EnsureAndSpendCreditForMessage 扣减 1 积分用于一条消息；不足时返回 ErrInsufficientCredits（应在设置 SSE 前调用）
func (s *ChatService) EnsureAndSpendCreditForMessage(userID uint) error {
	if s.walletRepo == nil {
		return nil
	}
	ok, err := s.walletRepo.HasEnoughCredits(userID, 1)
	if err != nil {
		return fmt.Errorf("check credits: %w", err)
	}
	if !ok {
		return ErrInsufficientCredits
	}
	return s.walletRepo.SpendCredits(userID, 1)
}

// StreamChat 发送用户消息、调用 LLM 流式返回，并写回 MongoDB；调用方需先扣积分（EnsureAndSpendCreditForMessage）；onChunk 用于 SSE 推送
func (s *ChatService) StreamChat(ctx context.Context, sessionID string, userID uint, userMessage string, persona *entity.ChatPersona, onChunk func(text string)) (assistantText string, err error) {
	if s.mongoChat != nil {
		if err := s.mongoChat.AppendMessage(ctx, sessionID, "user", userMessage); err != nil {
			return "", fmt.Errorf("append user message: %w", err)
		}
	}

	_, sess, err := s.GetOrCreateSession(ctx, sessionID, userID)
	if err != nil {
		return "", err
	}
	messages := sess.Messages
	if messages == nil {
		messages = []store.ChatMessage{}
	}
	messages = append(messages, store.ChatMessage{Role: "user", Content: userMessage})
	prompt := s.BuildPrompt(persona, messages)

	if s.llmClient == nil {
		assistantText = "(LLM not configured)"
		if onChunk != nil {
			onChunk(assistantText)
		}
		if s.mongoChat != nil {
			_ = s.mongoChat.AppendMessage(ctx, sessionID, "assistant", assistantText)
		}
		return assistantText, nil
	}

	assistantText, err = s.llmClient.StreamCompletion(ctx, prompt, onChunk)
	if err != nil {
		return "", err
	}
	if s.mongoChat != nil {
		_ = s.mongoChat.AppendMessage(ctx, sessionID, "assistant", assistantText)
	}
	return assistantText, nil
}
