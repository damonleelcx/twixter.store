// Package middleware 提供 viewing token 加解密（URL/cookie 仅存加密值，后端解密）。
// 实现委托给 backend/viewingtoken 以避免 service -> middleware 的 import cycle。
package middleware

import "backend/viewingtoken"

// EncryptViewingToken 将 viewing_light/viewing_dark 加密为可放在 URL/cookie 的 base64 字符串；密钥未配置时返回空
func EncryptViewingToken(plain string) string {
	return viewingtoken.EncryptViewingToken(plain)
}

// DecryptViewingToken 解密 URL/cookie 中的 token，返回 viewing_light 或 viewing_dark；失败或明文不合法则返回空
func DecryptViewingToken(token string) string {
	return viewingtoken.DecryptViewingToken(token)
}
