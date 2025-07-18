package sdk

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/hang666/EasyUKey/sdk/errs"
	"github.com/hang666/EasyUKey/sdk/request"
	"github.com/hang666/EasyUKey/sdk/response"
)

// AuthHelper 认证助手
type AuthHelper struct {
	client *Client
}

// NewAuthHelper 创建认证助手
func NewAuthHelper(client *Client) *AuthHelper {
	return &AuthHelper{client: client}
}

// GenerateChallenge 生成随机挑战码
func (h *AuthHelper) GenerateChallenge() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32

	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("%w: %v", errs.ErrRandomGenerationFailed, err)
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// SimpleAuth 简单认证流程
func (h *AuthHelper) SimpleAuth(username, apiKey string) (*response.VerifyAuthData, error) {
	// 生成挑战码
	challenge, err := h.GenerateChallenge()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrChallengeGenerationFailed, err)
	}

	// 发起认证
	authReq := &request.AuthRequest{
		UserID:    username,
		Challenge: challenge,
		Timeout:   60,
	}

	authData, err := h.client.StartAuth(username, authReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrAuthStartFailed, err)
	}

	// 等待认证完成
	return h.WaitForAuth(apiKey, authData.SessionID, 60*time.Second)
}

// WaitForAuth 等待认证完成
func (h *AuthHelper) WaitForAuth(apiKey, sessionID string, timeout time.Duration) (*response.VerifyAuthData, error) {
	verifyReq := &request.VerifyAuthRequest{
		SessionID: sessionID,
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if time.Now().After(deadline) {
			return nil, errs.ErrAuthTimeout
		}

		result, err := h.client.VerifyAuth(verifyReq)
		if err != nil {
			// 继续等待
			continue
		}

		if result.Success {
			return result, nil
		}
	}
	return nil, errs.ErrAuthTimeout
}

// QuickAuth 快速认证（带消息和动作）
func (h *AuthHelper) QuickAuth(username, apiKey, action, message string) (*response.VerifyAuthData, error) {
	challenge, err := h.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	authReq := &request.AuthRequest{
		UserID:    username,
		Challenge: challenge,
		Action:    action,
		Message:   message,
		Timeout:   60,
	}

	authData, err := h.client.StartAuth(username, authReq)
	if err != nil {
		return nil, err
	}

	return h.WaitForAuth(apiKey, authData.SessionID, 60*time.Second)
}
