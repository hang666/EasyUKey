package confirmation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// AuthRequest 认证请求结构, 这是给用户看的模型
type AuthRequest struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Challenge string    `json:"challenge"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthConfirmation 认证确认结果
type AuthConfirmation struct {
	RequestID string    `json:"request_id"`
	Confirmed bool      `json:"confirmed"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	confirmChan chan AuthConfirmation
	serverPort  int
)

// Init initializes the confirmation manager with the http server port.
func Init(port int) {
	confirmChan = make(chan AuthConfirmation, 1)
	serverPort = port
}

// ShowAuthRequest 显示认证请求（打开浏览器）
func ShowAuthRequest(request *AuthRequest) error {
	// 将请求序列化为JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("无法序列化请求: %w", err)
	}

	// Base64编码
	encodedData := base64.URLEncoding.EncodeToString(jsonData)

	url := fmt.Sprintf("http://localhost:%d?request=%s", serverPort, encodedData)
	return OpenBrowser(url)
}

// WaitForConfirmation 等待用户确认
func WaitForConfirmation(timeout time.Duration) (AuthConfirmation, error) {
	select {
	case confirmation := <-confirmChan:
		return confirmation, nil
	case <-time.After(timeout):
		return AuthConfirmation{}, fmt.Errorf("认证超时")
	}
}

// SendConfirmation 发送认证确认结果
func SendConfirmation(confirmation AuthConfirmation) {
	select {
	case confirmChan <- confirmation:
		// 成功发送
	default:
		// 通道阻塞或关闭，可能认证已超时
	}
}
