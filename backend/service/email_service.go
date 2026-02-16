package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// EmailService 邮件发送服务接口（用于密码重置等）
type EmailService interface {
	// SendPasswordResetEmail 发送密码重置邮件
	SendPasswordResetEmail(to, resetLink string) error
}

// emailService SMTP 实现
type emailService struct {
	host               string
	port               int
	user               string
	password           string
	from               string
	useTLS             bool   // true: 465 隐式 TLS；false: 587 STARTTLS
	tlsServerName      string // 可选：TLS 校验用的 ServerName（连接用 host 时若证书无 IP SAN 可设此项）
	insecureSkipVerify bool   // 可选：跳过 TLS 证书校验（仅内网/自签证书时使用）
}

// NewEmailService 从环境变量创建邮件服务。
// 需要: SMTP_HOST, SMTP_USER, SMTP_PASSWORD
// 可选: SMTP_PORT (默认 465), SMTP_FROM (默认 SMTP_USER), SMTP_USE_TLS (1/true 表示 465 隐式 TLS)
//       SMTP_TLS_SERVER_NAME (TLS 校验用的主机名，当 SMTP_HOST 为 IP 且证书无 IP SAN 时设置)
//       SMTP_INSECURE_SKIP_VERIFY (1/true 跳过 TLS 证书校验，仅内网/自签证书时使用)
// 若 SMTP_HOST 为空则返回 (nil, nil)，表示未配置邮件。
func NewEmailService() (EmailService, error) {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		return nil, nil
	}
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	password := os.Getenv("SMTP_PASSWORD")
	if user == "" || password == "" {
		return nil, fmt.Errorf("SMTP_USER and SMTP_PASSWORD are required when SMTP_HOST is set")
	}
	portStr := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if portStr == "" {
		portStr = "465"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("invalid SMTP_PORT: %s", portStr)
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = user
	}
	useTLS := false
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("SMTP_USE_TLS"))); v == "1" || v == "true" || v == "yes" {
		useTLS = true
	}
	tlsServerName := strings.TrimSpace(os.Getenv("SMTP_TLS_SERVER_NAME"))
	insecureSkipVerify := false
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("SMTP_INSECURE_SKIP_VERIFY"))); v == "1" || v == "true" || v == "yes" {
		insecureSkipVerify = true
	}
	return &emailService{
		host:               host,
		port:               port,
		user:               user,
		password:           password,
		from:               from,
		useTLS:             useTLS,
		tlsServerName:      tlsServerName,
		insecureSkipVerify: insecureSkipVerify,
	}, nil
}

// buildTLSConfig 构建 TLS 配置：ServerName 优先用 SMTP_TLS_SERVER_NAME，否则用 host；可选跳过证书校验。
func (e *emailService) buildTLSConfig() *tls.Config {
	serverName := e.host
	if e.tlsServerName != "" {
		serverName = e.tlsServerName
	}
	return &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: e.insecureSkipVerify,
	}
}

// SendPasswordResetEmail 发送密码重置邮件
func (e *emailService) SendPasswordResetEmail(to, resetLink string) error {
	subject := "Reset your password - Twixter"
	body := fmt.Sprintf("You requested a password reset. Click the link below to set a new password (valid for 1 hour):\n\n%s\n\nIf you did not request this, you can ignore this email.\n", resetLink)
	msg := "From: " + e.from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + body

	addr := net.JoinHostPort(e.host, strconv.Itoa(e.port))
	auth := smtp.PlainAuth("", e.user, e.password, e.host)

	const dialTimeout = 15 * time.Second
	tlsConfig := e.buildTLSConfig()
	if e.useTLS {
		// 隐式 TLS (如 465)
		dialer := &net.Dialer{Timeout: dialTimeout}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, e.host)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		if err = client.Mail(e.from); err != nil {
			return fmt.Errorf("smtp mail: %w", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		if _, err = w.Write([]byte(msg)); err != nil {
			return fmt.Errorf("smtp write: %w", err)
		}
		if err = w.Close(); err != nil {
			return fmt.Errorf("smtp close: %w", err)
		}
		return nil
	}

	// 587 STARTTLS
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(e.from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return nil
}

// noopEmailService 未配置邮件时的空实现
type noopEmailService struct{}

func (noopEmailService) SendPasswordResetEmail(to, resetLink string) error {
	return nil
}

// NoopEmailService 返回不发送邮件的占位实现
func NoopEmailService() EmailService {
	return noopEmailService{}
}
