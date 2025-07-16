package pin

import (
	"fmt"
	"time"
)

// PINManager PIN管理器，负责PIN的临时传递和超时控制
type PINManager struct {
	pinChan chan string
	timeout time.Duration
}

// NewPINManager 创建新的PIN管理器
func NewPINManager() *PINManager {
	return &PINManager{
		pinChan: make(chan string, 1),
		timeout: 60 * time.Second,
	}
}

// SendPIN 发送PIN到通道
func (pm *PINManager) SendPIN(pin string) {
	select {
	case pm.pinChan <- pin:
		// PIN发送成功
	default:
		// 通道已满，丢弃之前的PIN
		select {
		case <-pm.pinChan:
		default:
		}
		pm.pinChan <- pin
	}
}

// WaitPIN 等待PIN输入，带超时控制
func (pm *PINManager) WaitPIN() (string, error) {
	select {
	case pin := <-pm.pinChan:
		return pin, nil
	case <-time.After(pm.timeout):
		return "", fmt.Errorf("PIN输入超时")
	}
}

// Close 关闭PIN管理器
func (pm *PINManager) Close() {
	close(pm.pinChan)
}

// ValidatePIN 验证PIN格式（6位数字）
func ValidatePIN(pin string) error {
	if len(pin) != 6 {
		return fmt.Errorf("PIN必须为6位数字")
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return fmt.Errorf("PIN只能包含数字")
		}
	}
	return nil
}
