// Package previewtoken 提供短期预览 token（用于匿名 slideshow 预览 HLS 限时片段）。
// Token 采用 AES-GCM 加密 JSON payload，密钥来自 PREVIEW_TOKEN_SECRET（优先）或 VIEWING_TOKEN_SECRET（兜底）。
package previewtoken

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

type Payload struct {
	FileID        uint     `json:"file_id"`
	ExpiresAtUnix int64    `json:"exp"`
	Segments      []string `json:"segs"`
}

func secretKey() []byte {
	secret := strings.TrimSpace(os.Getenv("PREVIEW_TOKEN_SECRET"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("VIEWING_TOKEN_SECRET"))
	}
	if secret == "" {
		return nil
	}
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

func Encrypt(p Payload) (string, error) {
	key := secretKey()
	if key == nil {
		return "", errors.New("preview token secret not configured")
	}
	if p.FileID == 0 || len(p.Segments) == 0 {
		return "", errors.New("invalid payload")
	}
	if p.ExpiresAtUnix <= time.Now().Unix() {
		return "", errors.New("token already expired")
	}
	plain, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	buf := append(nonce, ciphertext...)
	return base64.URLEncoding.EncodeToString(buf), nil
}

func Decrypt(token string) (Payload, error) {
	var out Payload
	token = strings.TrimSpace(token)
	if token == "" {
		return out, errors.New("empty token")
	}
	key := secretKey()
	if key == nil {
		return out, errors.New("preview token secret not configured")
	}
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		raw, err = base64.StdEncoding.DecodeString(token)
		if err != nil {
			return out, errors.New("invalid token encoding")
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return out, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return out, err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns+1 {
		return out, errors.New("invalid token length")
	}
	nonce := raw[:ns]
	ciphertext := raw[ns:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return out, errors.New("invalid token")
	}
	if err := json.Unmarshal(plain, &out); err != nil {
		return out, errors.New("invalid token payload")
	}
	if out.FileID == 0 || len(out.Segments) == 0 {
		return out, errors.New("invalid token payload")
	}
	if out.ExpiresAtUnix <= time.Now().Unix() {
		return out, errors.New("token expired")
	}
	return out, nil
}

