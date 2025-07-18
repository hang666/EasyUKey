package identity

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hang666/EasyUKey/shared/pkg/errs"
)

const EncryptedFileExt = ".enc"

// Store 使用PIN+EncryptKey加密并存储数据
func Store(pin, encryptKey, keyType string, data []byte, basePath string) error {
	if pin == "" || encryptKey == "" || len(data) == 0 {
		return errs.ErrPINOrKeyEmpty
	}

	os.MkdirAll(basePath, 0o700)

	key := []byte(md5Hash(pin + "_" + encryptKey))
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return err
	}

	encrypted, err := encryptor.Encrypt(data)
	if err != nil {
		return err
	}

	filename := getKeyFilePath(keyType, basePath)
	return os.WriteFile(filename, encrypted, 0o600)
}

// Load 使用PIN+EncryptKey解密并加载数据
func Load(pin, encryptKey, keyType string, basePath string) ([]byte, error) {
	if pin == "" || encryptKey == "" {
		return nil, errs.ErrPINOrKeyEmpty
	}

	filename := getKeyFilePath(keyType, basePath)
	encrypted, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	key := []byte(md5Hash(pin + "_" + encryptKey))
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return nil, err
	}

	return encryptor.Decrypt(encrypted)
}

// KeyExists 检查密钥是否存在
func KeyExists(keyType string, basePath string) bool {
	filename := getKeyFilePath(keyType, basePath)
	_, err := os.Stat(filename)
	return err == nil
}

// SetTOTPSecret 存储TOTP密钥
func SetTOTPSecret(pin, encryptKey, secret, basePath string) error {
	return Store(pin, encryptKey, "totp", []byte(secret), basePath)
}

// SetOnceKey 存储一次性密钥
func SetOnceKey(pin, encryptKey, key, basePath string) error {
	return Store(pin, encryptKey, "once", []byte(key), basePath)
}

// GetTOTPSecret 获取TOTP密钥
func GetTOTPSecret(pin, encryptKey, basePath string) (string, error) {
	data, err := Load(pin, encryptKey, "totp", basePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetOnceKey 获取一次性密钥
func GetOnceKey(pin, encryptKey, basePath string) (string, error) {
	data, err := Load(pin, encryptKey, "once", basePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetFullKey 生成完整认证密钥
func GetFullKey(pin, encryptKey, serialNumber, volumeSerialNumber, basePath string) (string, error) {
	onceKey, err := GetOnceKey(pin, encryptKey, basePath)
	if err != nil {
		return "", err
	}

	totpURI, err := GetTOTPSecret(pin, encryptKey, basePath)
	if err != nil {
		return "", err
	}

	totpConfig, err := ParseTOTPURI(totpURI)
	if err != nil {
		return "", err
	}

	totpCode, err := GenerateTOTPCode(totpConfig, time.Now())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:_:%s:_:%s:_:%s", onceKey, totpCode, serialNumber, volumeSerialNumber), nil
}

// SaveInitialKeys 保存初始化密钥
func SaveInitialKeys(pin, encryptKey, onceKey, totpURI, basePath string) error {
	if err := SetOnceKey(pin, encryptKey, onceKey, basePath); err != nil {
		return err
	}
	return SetTOTPSecret(pin, encryptKey, totpURI, basePath)
}

// IsInitialized 检查设备是否已经初始化
func IsInitialized(basePath string) bool {
	return KeyExists("totp", basePath)
}

// getKeyFilePath 获取密钥文件路径
func getKeyFilePath(keyType string, basePath string) string {
	var filename string
	switch keyType {
	case "totp":
		filename = "t.dat" + EncryptedFileExt
	case "once":
		filename = "o.dat" + EncryptedFileExt
	default:
		filename = fmt.Sprintf("%s.dat%s", keyType, EncryptedFileExt)
	}
	return filepath.Join(basePath, filename)
}

// md5Hash 计算MD5哈希
func md5Hash(data string) string {
	hash := md5.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}
