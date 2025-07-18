package errs

import "errors"

// 客户端核心错误
var (
	// USB设备错误
	ErrUSBDeviceNotFound      = errors.New("USB设备未找到")
	ErrUSBCommunicationFailed = errors.New("USB设备通信失败")

	// UI错误
	ErrBrowserLaunchFailed    = errors.New("浏览器启动失败")
	ErrUserInteractionTimeout = errors.New("用户交互超时")

	// 密钥错误
	ErrKeyStorageCorrupted = errors.New("密钥存储损坏")
	ErrKeyGenerationFailed = errors.New("密钥生成失败")
)
