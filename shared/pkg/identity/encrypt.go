package identity

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Encryptor 封装加解密逻辑
type Encryptor struct {
	Key []byte // 32字节（256位）密钥
}

// NewEncryptor 创建一个新的加解密器
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}
	return &Encryptor{Key: key}, nil
}

// Encrypt 加密数据
func (e *Encryptor) Encrypt(plainData []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, err
	}

	// 初始化向量 IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// PKCS7 Padding
	plainData = pkcs7Pad(plainData, aes.BlockSize)

	cipherText := make([]byte, len(plainData))

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, plainData)

	// 返回 IV + 密文
	return append(iv, cipherText...), nil
}

// Decrypt 解密数据
func (e *Encryptor) Decrypt(cipherData []byte) ([]byte, error) {
	if len(cipherData) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, err
	}

	iv := cipherData[:aes.BlockSize]
	cipherText := cipherData[aes.BlockSize:]

	if len(cipherText)%aes.BlockSize != 0 {
		return nil, errors.New("cipherText is not a multiple of the block size")
	}

	plainText := make([]byte, len(cipherText))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plainText, cipherText)

	// 去除 Padding
	return pkcs7Unpad(plainText)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("data is empty")
	}
	padLen := int(data[length-1])
	if padLen > length || padLen > aes.BlockSize {
		return nil, errors.New("invalid padding size")
	}
	for _, v := range data[length-padLen:] {
		if int(v) != padLen {
			return nil, errors.New("invalid padding content")
		}
	}
	return data[:length-padLen], nil
}
