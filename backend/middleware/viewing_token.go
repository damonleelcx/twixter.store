// Package middleware 提供 viewing token 加解密（URL/cookie 仅存加密值，后端解密）。
// 需设置环境变量 VIEWING_TOKEN_SECRET（任意长度，SHA256 派生密钥）；未设置时加密不可用，仅兼容旧明文/base64 token。
package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// getViewingSecretKey 从环境变量 VIEWING_TOKEN_SECRET 派生 32 字节密钥；若未设置或太短则返回 nil
func getViewingSecretKey() []byte {
	secret := os.Getenv("VIEWING_TOKEN_SECRET")
	if secret == "" {
		return nil
	}
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

// EncryptViewingToken 将 viewing_light/viewing_dark 加密为可放在 URL/cookie 的 base64 字符串；密钥未配置时返回空
func EncryptViewingToken(plain string) string {
	plain = strings.TrimSpace(plain)
	if plain != "viewing_light" && plain != "viewing_dark" {
		return ""
	}
	key := getViewingSecretKey()
	if key == nil {
		return ""
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return ""
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	// 格式: nonceLen(8) + nonce + ciphertext，再 base64
	buf := make([]byte, 8+len(nonce)+len(ciphertext))
	binary.BigEndian.PutUint64(buf[0:8], uint64(len(nonce)))
	copy(buf[8:8+len(nonce)], nonce)
	copy(buf[8+len(nonce):], ciphertext)
	return base64.URLEncoding.EncodeToString(buf)
}

// DecryptViewingToken 解密 URL/cookie 中的 token，返回 viewing_light 或 viewing_dark；失败或明文不合法则返回空
// 兼容旧格式：明文 viewing_light/viewing_dark 或 base64(明文) 仍可解析
func DecryptViewingToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	// 兼容：直接是明文
	if token == "viewing_light" || token == "viewing_dark" {
		return token
	}
	// 兼容：base64 编码的明文（旧格式）
	if decoded, err := base64.StdEncoding.DecodeString(token); err == nil {
		s := strings.TrimSpace(string(decoded))
		if s == "viewing_light" || s == "viewing_dark" {
			return s
		}
	}
	// 新格式：base64url( nonceLen(8) + nonce + ciphertext )
	key := getViewingSecretKey()
	if key == nil {
		return ""
	}
	raw, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		// 尝试标准 base64
		raw, err = base64.StdEncoding.DecodeString(token)
		if err != nil || len(raw) < 8 {
			return ""
		}
	}
	if len(raw) < 8 {
		return ""
	}
	nonceLen := binary.BigEndian.Uint64(raw[0:8])
	if nonceLen > 32 || int(nonceLen)+8 > len(raw) {
		return ""
	}
	nonce := raw[8 : 8+nonceLen]
	ciphertext := raw[8+nonceLen:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(plain))
	if s == "viewing_light" || s == "viewing_dark" {
		return s
	}
	return ""
}
