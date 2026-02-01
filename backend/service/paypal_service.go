package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	paypalSandboxURL = "https://api-m.sandbox.paypal.com"
	paypalLiveURL    = "https://api-m.paypal.com"
)

// PayPalService PayPal 支付服务接口（备用支付）
type PayPalService interface {
	// CreateOrder 创建订单，返回 PayPal 订单 ID
	CreateOrder(amountUSD string, currency, description string) (orderID string, err error)
	// CaptureOrder 捕获订单完成支付
	CaptureOrder(orderID string) (captureID string, err error)
}

type paypalService struct {
	baseURL      string
	clientID     string
	clientSecret string
	client       *http.Client
	mu           sync.Mutex
	accessToken  string
	tokenExpiry  time.Time
}

// NewPayPalService 创建 PayPal 服务实例（可选，未配置时返回 nil）
func NewPayPalService() (PayPalService, error) {
	clientID := os.Getenv("PAYPAL_CLIENT_ID")
	clientSecret := os.Getenv("PAYPAL_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("PAYPAL_CLIENT_ID or PAYPAL_CLIENT_SECRET not set")
	}

	baseURL := paypalLiveURL
	if os.Getenv("PAYPAL_MODE") == "sandbox" || os.Getenv("PAYPAL_MODE") == "" {
		baseURL = paypalSandboxURL
	}

	return &paypalService{
		baseURL:      baseURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 30 * time.Second},
		accessToken:  "",
		tokenExpiry:  time.Time{},
	}, nil
}

// getAccessToken 获取或刷新 OAuth 访问令牌
func (p *paypalService) getAccessToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return p.accessToken, nil
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/v1/oauth2/token", bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(p.clientID+":"+p.clientSecret)))

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("paypal oauth failed: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	p.accessToken = result.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return p.accessToken, nil
}

// CreateOrder 创建 PayPal 订单
func (p *paypalService) CreateOrder(amountUSD string, currency, description string) (orderID string, err error) {
	token, err := p.getAccessToken()
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]interface{}{
					"currency_code": currency,
					"value":         amountUSD,
				},
				"description": description,
			},
		},
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/v2/checkout/orders", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal create order failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// 尝试从 Location 头获取（Prefer: return=minimal 时 body 可能为空）
		if loc := resp.Header.Get("Location"); loc != "" {
			for i := len(loc) - 1; i >= 0; i-- {
				if loc[i] == '/' {
					return loc[i+1:], nil
				}
			}
		}
		return "", err
	}
	if result.ID == "" {
		if loc := resp.Header.Get("Location"); loc != "" {
			for i := len(loc) - 1; i >= 0; i-- {
				if loc[i] == '/' {
					return loc[i+1:], nil
				}
			}
		}
		return "", fmt.Errorf("paypal create order: no order id in response")
	}
	return result.ID, nil
}

// CaptureOrder 捕获订单
func (p *paypalService) CaptureOrder(orderID string) (captureID string, err error) {
	token, err := p.getAccessToken()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/v2/checkout/orders/"+orderID+"/capture", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Prefer", "return=minimal")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal capture failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	_ = json.Unmarshal(respBody, &result)
	if len(result.PurchaseUnits) > 0 && len(result.PurchaseUnits[0].Payments.Captures) > 0 {
		return result.PurchaseUnits[0].Payments.Captures[0].ID, nil
	}
	if result.ID != "" {
		return result.ID, nil
	}
	return orderID, nil
}
