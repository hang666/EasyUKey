package callback

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hang666/EasyUKey/shared/pkg/messages"
)

// GenerateSignature 生成回调签名
func GenerateSignature(req *messages.CallbackRequest, secret string) string {
	// 构建签名字符串
	data := map[string]string{
		"session_id": req.SessionID,
		"user_id":    req.UserID,
		"status":     req.Status,
		"challenge":  req.Challenge,
		"action":     req.Action,
		"device_id":  strconv.FormatUint(uint64(req.DeviceID), 10),
		"timestamp":  strconv.FormatInt(req.Timestamp, 10),
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
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证回调签名
func VerifySignature(req *messages.CallbackRequest, secret string) bool {
	originalSignature := req.Signature
	req.Signature = "" // 临时清空签名字段

	expectedSignature := GenerateSignature(req, secret)

	req.Signature = originalSignature // 恢复原始签名
	return hmac.Equal([]byte(originalSignature), []byte(expectedSignature))
}

// ValidateCallbackRequest 验证回调请求
func ValidateCallbackRequest(req *messages.CallbackRequest, secret string) error {
	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.Status == "" {
		return fmt.Errorf("status is required")
	}
	if req.Challenge == "" {
		return fmt.Errorf("challenge is required")
	}
	if req.Timestamp == 0 {
		return fmt.Errorf("timestamp is required")
	}
	if req.Signature == "" {
		return fmt.Errorf("signature is required")
	}

	// 验证签名
	if !VerifySignature(req, secret) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
