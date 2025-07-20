package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/hang666/EasyUKey/shared/pkg/identity"
)

// GenerateAuthToken 生成新格式的认证token
// 格式: {challenge}:{totpCode}:{authToken}
// authToken = HMAC-SHA256(challenge + onceKey + serialNumber + volumeSerialNumber, encryptionKey)
func GenerateAuthToken(challenge, pin, encryptKeyStr, serialNumber, volumeSerialNumber, basePath string) (string, error) {
	// 获取OnceKey
	onceKey, err := identity.GetOnceKey(pin, encryptKeyStr, basePath)
	if err != nil {
		return "", fmt.Errorf("获取OnceKey失败: %w", err)
	}

	// 获取TOTP密钥并生成代码
	totpURI, err := identity.GetTOTPSecret(pin, encryptKeyStr, basePath)
	if err != nil {
		return "", fmt.Errorf("获取TOTP密钥失败: %w", err)
	}

	totpConfig, err := identity.ParseTOTPURI(totpURI)
	if err != nil {
		return "", fmt.Errorf("解析TOTP配置失败: %w", err)
	}

	totpCode, err := identity.GenerateTOTPCode(totpConfig, time.Now())
	if err != nil {
		return "", fmt.Errorf("生成TOTP代码失败: %w", err)
	}

	// 生成authToken（不包含totpCode）
	authToken := generateHMACToken(challenge, onceKey, serialNumber, volumeSerialNumber, encryptKeyStr)

	return fmt.Sprintf("%s:%s:%s", challenge, totpCode, authToken), nil
}

// ValidateAuthToken 验证认证token
func ValidateAuthToken(fullToken, expectedChallenge, onceKey, totpSecret, serialNumber, volumeSerialNumber, encryptKeyStr string) error {
	// 解析token格式 challenge:totpCode:authToken
	parts := strings.Split(fullToken, ":")
	if len(parts) != 3 {
		return fmt.Errorf("认证token格式无效，期望3段，实际%d段", len(parts))
	}

	challenge := parts[0]
	totpCode := parts[1]
	authToken := parts[2]

	// 验证挑战码
	if challenge != expectedChallenge {
		return fmt.Errorf("挑战码不匹配")
	}

	// 验证TOTP代码
	totpConfig, err := identity.ParseTOTPURI(totpSecret)
	if err != nil {
		return fmt.Errorf("解析TOTP配置失败: %w", err)
	}

	isValidTOTP, err := identity.VerifyTOTPCode(totpConfig, totpCode, time.Now())
	if err != nil {
		return fmt.Errorf("验证TOTP代码失败: %w", err)
	}

	if !isValidTOTP {
		return fmt.Errorf("TOTP验证码无效")
	}

	// 验证authToken
	expectedToken := generateHMACToken(challenge, onceKey, serialNumber, volumeSerialNumber, encryptKeyStr)
	if !hmac.Equal([]byte(authToken), []byte(expectedToken)) {
		return fmt.Errorf("认证token验证失败")
	}

	return nil
}

// generateHMACToken 生成HMAC-SHA256 token
func generateHMACToken(challenge, onceKey, serialNumber, volumeSerialNumber, encryptKeyStr string) string {
	// 组合认证数据
	data := challenge + onceKey + serialNumber + volumeSerialNumber

	// 使用encryptKeyStr作为HMAC密钥
	h := hmac.New(sha256.New, []byte(encryptKeyStr))
	h.Write([]byte(data))

	return hex.EncodeToString(h.Sum(nil))
}
