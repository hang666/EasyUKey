package identity

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

// ECDHKeyPair 椭圆曲线 Diffie-Hellman 密钥对
type ECDHKeyPair struct {
	PrivateKey *ecdh.PrivateKey
	PublicKey  *ecdh.PublicKey
}

// KeyExchange 密钥交换器
type KeyExchange struct {
	keyPair   *ECDHKeyPair
	sharedKey []byte
}

// NewKeyExchange 创建新的密钥交换器
func NewKeyExchange() (*KeyExchange, error) {
	keyPair, err := GenerateECDHKeyPair()
	if err != nil {
		return nil, err
	}
	return &KeyExchange{
		keyPair: keyPair,
	}, nil
}

// GenerateECDHKeyPair 生成 ECDH 密钥对 (使用 P-256 曲线)
func GenerateECDHKeyPair() (*ECDHKeyPair, error) {
	curve := ecdh.P256()
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ECDHKeyPair{
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
	}, nil
}

// GetPublicKeyBytes 获取公钥字节
func (kp *ECDHKeyPair) GetPublicKeyBytes() []byte {
	return kp.PublicKey.Bytes()
}

// GetPublicKeyBase64 获取公钥的Base64编码
func (kp *ECDHKeyPair) GetPublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.GetPublicKeyBytes())
}

// GetPublicKeyBase64 获取本地公钥的Base64编码
func (kx *KeyExchange) GetPublicKeyBase64() string {
	return kx.keyPair.GetPublicKeyBase64()
}

// ComputeSharedKey 计算共享密钥
func (kx *KeyExchange) ComputeSharedKey(peerPublicKeyBase64 string) error {
	// 解码对方的公钥
	peerPublicKeyBytes, err := base64.StdEncoding.DecodeString(peerPublicKeyBase64)
	if err != nil {
		return err
	}

	// 创建对方的公钥对象
	curve := ecdh.P256()
	peerPublicKey, err := curve.NewPublicKey(peerPublicKeyBytes)
	if err != nil {
		return err
	}

	// 计算共享密钥
	sharedSecret, err := kx.keyPair.PrivateKey.ECDH(peerPublicKey)
	if err != nil {
		return err
	}

	// 使用 SHA-256 派生会话密钥
	hash := sha256.Sum256(sharedSecret)
	kx.sharedKey = hash[:]

	return nil
}

// GetSharedKey 获取共享密钥
func (kx *KeyExchange) GetSharedKey() ([]byte, error) {
	if kx.sharedKey == nil {
		return nil, errors.New("shared key not computed")
	}
	return kx.sharedKey, nil
}

// CreateEncryptor 创建基于共享密钥的加密器
func (kx *KeyExchange) CreateEncryptor() (*Encryptor, error) {
	sharedKey, err := kx.GetSharedKey()
	if err != nil {
		return nil, err
	}
	return NewEncryptor(sharedKey)
}

// EncryptGCM 使用AES-GCM加密数据
func (e *Encryptor) EncryptGCM(plainData []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, nil, err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	// 生成随机的nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	// 加密数据
	cipherText := gcm.Seal(nil, nonce, plainData, nil)

	return cipherText, nonce, nil
}

// DecryptGCM 使用AES-GCM解密数据
func (e *Encryptor) DecryptGCM(cipherData []byte, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return nil, err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 解密数据
	plainText, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, err
	}

	return plainText, nil
}

// EncryptMessage 加密消息并返回Base64编码的密文和nonce
func (e *Encryptor) EncryptMessage(plainText []byte) (cipherBase64 string, nonceBase64 string, err error) {
	cipherData, nonce, err := e.EncryptGCM(plainText)
	if err != nil {
		return "", "", err
	}

	cipherBase64 = base64.StdEncoding.EncodeToString(cipherData)
	nonceBase64 = base64.StdEncoding.EncodeToString(nonce)

	return cipherBase64, nonceBase64, nil
}

// DecryptMessage 解密Base64编码的消息
func (e *Encryptor) DecryptMessage(cipherBase64 string, nonceBase64 string) ([]byte, error) {
	cipherData, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceBase64)
	if err != nil {
		return nil, err
	}

	return e.DecryptGCM(cipherData, nonce)
}
