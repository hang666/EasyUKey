package identity

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// EncryptedFileExt 加密文件扩展名
	EncryptedFileExt = ".enc"
)

// SecureStorage 安全存储管理器
type SecureStorage struct {
	encryptor *Encryptor
	basePath  string
}

// NewSecureStorage 创建安全存储实例
func NewSecureStorage(password string, basePath string) (*SecureStorage, error) {
	if password == "" {
		return nil, errors.New("密码不能为空")
	}

	// 确保目录存在
	if err := os.MkdirAll(basePath, 0o700); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 使用MD5哈希作为加密密钥（与现有代码保持一致）
	key := []byte(md5Hash(password))

	// 创建加密器
	encryptor, err := NewEncryptor(key)
	if err != nil {
		return nil, fmt.Errorf("创建加密器失败: %w", err)
	}

	return &SecureStorage{
		encryptor: encryptor,
		basePath:  basePath,
	}, nil
}

// StoreKey 安全存储密钥
func (s *SecureStorage) StoreKey(keyType string, data []byte) error {
	if len(data) == 0 {
		return errors.New("数据不能为空")
	}

	// 使用现有的加密器加密数据
	encrypted, err := s.encryptor.Encrypt(data)
	if err != nil {
		return fmt.Errorf("加密数据失败: %w", err)
	}

	// 写入文件
	filename := s.getKeyFilePath(keyType)
	if err := os.WriteFile(filename, encrypted, 0o600); err != nil {
		return fmt.Errorf("写入加密文件失败: %w", err)
	}

	return nil
}

// LoadKey 安全读取密钥
func (s *SecureStorage) LoadKey(keyType string) ([]byte, error) {
	filename := s.getKeyFilePath(keyType)

	// 读取加密文件
	encrypted, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取加密文件失败: %w", err)
	}

	// 使用现有的解密器解密数据
	decrypted, err := s.encryptor.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("解密数据失败: %w", err)
	}

	return decrypted, nil
}

// KeyExists 检查密钥是否存在
func (s *SecureStorage) KeyExists(keyType string) bool {
	filename := s.getKeyFilePath(keyType)
	_, err := os.Stat(filename)
	return err == nil
}

// DeleteKey 删除密钥
func (s *SecureStorage) DeleteKey(keyType string) error {
	filename := s.getKeyFilePath(keyType)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除密钥文件失败: %w", err)
	}
	return nil
}

// ListKeys 列出所有存储的密钥类型
func (s *SecureStorage) ListKeys() ([]string, error) {
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	var keys []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if filepath.Ext(name) == EncryptedFileExt {
			keyType := name[:len(name)-len(EncryptedFileExt)]
			keys = append(keys, keyType)
		}
	}

	return keys, nil
}

// getKeyFilePath 获取密钥文件路径，按照现有命名规则
func (s *SecureStorage) getKeyFilePath(keyType string) string {
	var filename string
	switch keyType {
	case "totp":
		filename = "t.dat" + EncryptedFileExt
	case "once":
		filename = "o.dat" + EncryptedFileExt
	default:
		filename = fmt.Sprintf("%s.dat%s", keyType, EncryptedFileExt)
	}
	return filepath.Join(s.basePath, filename)
}

// md5Hash 计算MD5哈希（与现有代码保持一致）
func md5Hash(data string) string {
	hash := md5.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}
