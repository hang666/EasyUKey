package errs

import "errors"

// 服务端核心错误
var (
	// 数据库错误
	ErrDatabaseConnection = errors.New("数据库连接失败")
	ErrDatabaseQuery      = errors.New("数据库查询错误")

	// 配置错误
	ErrConfigInvalid = errors.New("配置文件格式错误")
	ErrConfigMissing = errors.New("配置文件不存在")

	// 加密错误
	ErrEncryptionFailed = errors.New("加密操作失败")
	ErrDecryptionFailed = errors.New("解密操作失败")

	// WebSocket错误
	ErrMessageValidation = errors.New("消息验证失败")
	ErrBroadcastFailed   = errors.New("广播消息失败")

	// 通用错误
	ErrInternal = errors.New("内部服务器错误")
)
