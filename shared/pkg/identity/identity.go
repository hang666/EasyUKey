package identity

import (
	"fmt"
	"time"
)

var secureStorage *SecureStorage

// InitSecureStorage 初始化安全存储
func InitSecureStorage(password string, basePath string) error {
	var err error
	secureStorage, err = NewSecureStorage(password, basePath)
	return err
}

// SetTOTPSecretSecure 存储TOTP密钥
func SetTOTPSecretSecure(secret string) error {
	if secureStorage == nil {
		return fmt.Errorf("安全存储未初始化")
	}
	return secureStorage.StoreKey("totp", []byte(secret))
}

// SetOnceKeySecure 存储一次性密钥
func SetOnceKeySecure(key string) error {
	if secureStorage == nil {
		return fmt.Errorf("安全存储未初始化")
	}
	return secureStorage.StoreKey("once", []byte(key))
}

// GetTOTPSecretSecure 获取TOTP密钥
func GetTOTPSecretSecure() (string, error) {
	if secureStorage == nil {
		return "", fmt.Errorf("安全存储未初始化")
	}
	data, err := secureStorage.LoadKey("totp")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetOnceKeySecure 获取一次性密钥
func GetOnceKeySecure() (string, error) {
	if secureStorage == nil {
		return "", fmt.Errorf("安全存储未初始化")
	}
	data, err := secureStorage.LoadKey("once")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetFullKeySecure 生成完整认证密钥
func GetFullKeySecure(serialNumber string, volumeSerialNumber string) (string, error) {
	if secureStorage == nil {
		return "", fmt.Errorf("安全存储未初始化")
	}

	onceKey, err := GetOnceKeySecure()
	if err != nil {
		return "", fmt.Errorf("获取一次性密钥失败: %w", err)
	}

	totpURI, err := GetTOTPSecretSecure()
	if err != nil {
		return "", fmt.Errorf("获取TOTP密钥失败: %w", err)
	}

	totpConfig, err := ParseTOTPURI(totpURI)
	if err != nil {
		return "", fmt.Errorf("解析TOTP URI失败: %w", err)
	}

	totpCode, err := GenerateTOTPCode(totpConfig, time.Now())
	if err != nil {
		return "", fmt.Errorf("生成TOTP代码失败: %w", err)
	}

	fullKey := fmt.Sprintf("%s:_:%s:_:%s:_:%s", onceKey, totpCode, serialNumber, volumeSerialNumber)
	return fullKey, nil
}

// SaveInitialKeys 保存初始化密钥
func SaveInitialKeys(onceKey, totpURI string) error {
	if err := SetOnceKeySecure(onceKey); err != nil {
		return fmt.Errorf("保存一次性密钥失败: %w", err)
	}
	if err := SetTOTPSecretSecure(totpURI); err != nil {
		return fmt.Errorf("保存TOTP密钥失败: %w", err)
	}
	return nil
}

// IsInitialized 检查设备是否已经初始化
func IsInitialized() bool {
	if secureStorage == nil {
		return false
	}
	return secureStorage.KeyExists("totp")
}
