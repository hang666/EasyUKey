package sdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hang666/EasyUKey/sdk/request"
)

// ValidateCallbackRequest 验证回调请求
func ValidateCallbackRequest(data []byte, secret string) (*request.CallbackRequest, error) {
	var req request.CallbackRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %v", err)
	}

	// 验证必需字段
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Status == "" {
		return nil, fmt.Errorf("status is required")
	}
	if req.Challenge == "" {
		return nil, fmt.Errorf("challenge is required")
	}
	if req.Timestamp == 0 {
		return nil, fmt.Errorf("timestamp is required")
	}
	if req.Signature == "" {
		return nil, fmt.Errorf("signature is required")
	}

	// 验证时间戳（防重放攻击）
	now := time.Now().Unix()
	if req.Timestamp < now-300 || req.Timestamp > now+300 { // 5分钟窗口
		return nil, fmt.Errorf("timestamp is out of acceptable range")
	}

	// 验证签名
	if !verifyCallbackSignature(&req, secret) {
		return nil, fmt.Errorf("invalid signature")
	}

	return &req, nil
}

// verifyCallbackSignature 验证回调签名
func verifyCallbackSignature(req *request.CallbackRequest, secret string) bool {
	originalSignature := req.Signature
	req.Signature = "" // 临时清空签名字段

	expectedSignature := generateCallbackSignature(req, secret)

	req.Signature = originalSignature // 恢复原始签名
	return hmac.Equal([]byte(originalSignature), []byte(expectedSignature))
}

// generateCallbackSignature 生成回调签名
func generateCallbackSignature(req *request.CallbackRequest, secret string) string {
	// 构建签名字符串
	data := map[string]string{
		"session_id": req.SessionID,
		"user_id":    req.UserID,
		"status":     req.Status,
		"challenge":  req.Challenge,
		"action":     req.Action,
		"device_id":  fmt.Sprintf("%d", req.DeviceID),
		"timestamp":  fmt.Sprintf("%d", req.Timestamp),
	}

	// 按字母顺序排序键值对
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建签名字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+data[k])
	}
	signString := strings.Join(parts, "&")

	// 计算HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signString))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

// CallbackHandler 回调处理器接口
type CallbackHandler interface {
	OnAuthSuccess(req *request.CallbackRequest) error
	OnAuthFailure(req *request.CallbackRequest) error
}

// HandleCallback 处理回调请求
func HandleCallback(data []byte, secret string, handler CallbackHandler) error {
	req, err := ValidateCallbackRequest(data, secret)
	if err != nil {
		return fmt.Errorf("回调验证失败: %v", err)
	}

	switch req.Status {
	case "success":
		return handler.OnAuthSuccess(req)
	case "failed":
		return handler.OnAuthFailure(req)
	default:
		return fmt.Errorf("unknown callback status: %s", req.Status)
	}
}
