package confirmation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
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

// AuthResult 认证结果结构
type AuthResult struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// AuthState 认证状态
type AuthState int

const (
	StateIdle       AuthState = iota // 空闲状态
	StateWaiting                     // 等待用户确认
	StateProcessing                  // 正在处理认证
	StateCompleted                   // 认证完成
)

var (
	confirmChan  chan AuthConfirmation
	resultChan   chan AuthResult
	serverPort   int
	currentState AuthState
	currentReqID string
	stateMutex   sync.RWMutex
)

// Init initializes the confirmation manager with the http server port.
func Init(port int) {
	confirmChan = make(chan AuthConfirmation, 1)
	resultChan = make(chan AuthResult, 1)
	serverPort = port
	currentState = StateIdle
}

// ShowAuthRequest 显示认证请求（打开浏览器）
func ShowAuthRequest(request *AuthRequest) error {
	stateMutex.Lock()
	currentState = StateWaiting
	currentReqID = request.ID
	stateMutex.Unlock()

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
		stateMutex.Lock()
		if currentState == StateWaiting {
			currentState = StateProcessing
		}
		stateMutex.Unlock()
		return confirmation, nil
	case <-time.After(timeout):
		stateMutex.Lock()
		currentState = StateIdle
		currentReqID = ""
		stateMutex.Unlock()
		return AuthConfirmation{}, fmt.Errorf("认证超时")
	}
}

// SendConfirmation 发送认证确认结果
func SendConfirmation(confirmation AuthConfirmation) {
	stateMutex.RLock()
	state := currentState
	reqID := currentReqID
	stateMutex.RUnlock()

	// 检查当前状态和请求ID
	if state != StateWaiting {
		// 如果不是等待状态，忽略此确认
		return
	}

	if reqID != confirmation.RequestID {
		// 请求ID不匹配，忽略此确认
		return
	}

	select {
	case confirmChan <- confirmation:
		// 成功发送
	default:
		// 通道阻塞或关闭，可能认证已超时
	}
}

// ShowPINSetupPage 显示PIN设置页面
func ShowPINSetupPage() error {
	url := fmt.Sprintf("http://localhost:%d/pin", serverPort)
	return OpenBrowser(url)
}

// resetState 重置认证状态的辅助函数
func resetState() {
	stateMutex.Lock()
	currentState = StateIdle
	currentReqID = ""
	stateMutex.Unlock()
}

// WaitForResult 等待认证结果
func WaitForResult(timeout time.Duration) (AuthResult, error) {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case result := <-resultChan:
			if time.Since(result.Timestamp) > 10*time.Second {
				continue // 数据已过期，继续等待
			}
			return result, nil
		case <-time.After(time.Until(deadline)):
			resetState()
			return AuthResult{}, fmt.Errorf("认证超时")
		}
	}
}

// SendResult 发送认证结果
func SendResult(success bool, message string) {
	result := AuthResult{
		Success:   success,
		Message:   message,
		Timestamp: time.Now(),
	}

	stateMutex.Lock()
	currentState = StateCompleted
	stateMutex.Unlock()

	select {
	case resultChan <- result:
		// 成功发送
	default:
		// 防止阻塞
	}
}

// GetCurrentState 获取当前认证状态
func GetCurrentState() (AuthState, string) {
	stateMutex.RLock()
	defer stateMutex.RUnlock()
	return currentState, currentReqID
}

// ResetState 重置认证状态
func ResetState() {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	currentState = StateIdle
	currentReqID = ""
}
