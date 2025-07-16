package errors

import "errors"

// 预定义的错误变量，避免重复创建
var (
	// 请求参数错误
	ErrInvalidRequest    = errors.New("请求参数格式错误")
	ErrMissingAPIKey     = errors.New("缺少API密钥")
	ErrMissingChallenge  = errors.New("缺少挑战码")
	ErrMissingSessionID  = errors.New("缺少会话ID")
	ErrMissingUsername   = errors.New("缺少用户名")
	ErrMissingName       = errors.New("缺少名称")
	ErrMissingDeviceInfo = errors.New("缺少设备标识信息")
	ErrMissingAdminKey   = errors.New("缺少管理员密钥")
	ErrInvalidKey        = errors.New("错误的密钥")
	ErrInvalidDeviceID   = errors.New("设备ID格式错误")

	// 认证与权限错误
	ErrAPIKeyInvalid    = errors.New("无效的API密钥")
	ErrPermissionDenied = errors.New("设备权限不足")

	// 业务实体错误
	ErrUserNotFound        = errors.New("用户不存在")
	ErrUserNotOnline       = errors.New("用户未在线")
	ErrUserAlreadyExists   = errors.New("用户名已存在")
	ErrDeviceNotFound      = errors.New("设备不存在")
	ErrDeviceAlreadyExists = errors.New("设备已存在")
	ErrDeviceNotActive     = errors.New("设备未激活")
	ErrDeviceAlreadyBound  = errors.New("设备已绑定用户")

	// 会话错误
	ErrSessionNotFound  = errors.New("认证会话不存在")
	ErrSessionExpired   = errors.New("认证会话已过期")
	ErrSessionCompleted = errors.New("认证会话已完成")

	// 认证流程错误
	ErrAuthInvalidKey       = errors.New("认证密钥无效")
	ErrAuthDeviceInactive   = errors.New("设备未激活")
	ErrAuthChallengeInvalid = errors.New("挑战码不匹配")
	ErrAuthSerialMismatch   = errors.New("设备序列号不匹配")
	ErrAuthOnceKeyMismatch  = errors.New("一次性密钥不匹配")
	ErrAuthTOTPInvalid      = errors.New("TOTP验证码无效")

	// 客户端错误
	ErrDeviceNotFoundClient = errors.New("设备未找到")
	ErrShowPageFailed       = errors.New("无法显示认证页面")
	ErrWaitConfirmFailed    = errors.New("等待用户确认失败")
	ErrGetOnceKeyFailed     = errors.New("获取当前OnceKey失败")
	ErrGetFullKeyFailed     = errors.New("获取完整密钥失败")
	ErrUserRejected         = errors.New("用户拒绝认证")

	// WebSocket错误
	ErrWSValidation       = errors.New("消息验证错误")
	ErrWSParse            = errors.New("消息解析错误")
	ErrWSUnknownMessage   = errors.New("未知消息类型")
	ErrWSChannelFull      = errors.New("发送通道已满")
	ErrWSConnectionClosed = errors.New("连接已关闭")
	ErrWSInternal         = errors.New("内部错误")
)

// HTTPStatus 错误对应的HTTP状态码映射
var HTTPStatus = map[error]int{
	// 400 Bad Request
	ErrInvalidRequest:      400,
	ErrMissingAPIKey:       400,
	ErrMissingChallenge:    400,
	ErrMissingSessionID:    400,
	ErrMissingUsername:     400,
	ErrMissingName:         400,
	ErrMissingDeviceInfo:   400,
	ErrMissingAdminKey:     400,
	ErrInvalidKey:          400,
	ErrInvalidDeviceID:     400,
	ErrDeviceAlreadyExists: 400,
	ErrDeviceNotActive:     400,
	ErrDeviceAlreadyBound:  400,
	ErrUserAlreadyExists:   400,
	ErrSessionExpired:      400,
	ErrSessionCompleted:    400,

	// 401 Unauthorized
	ErrAPIKeyInvalid: 401,

	// 403 Forbidden
	ErrPermissionDenied: 403,

	// 404 Not Found
	ErrUserNotFound:    404,
	ErrDeviceNotFound:  404,
	ErrSessionNotFound: 404,

	// 503 Service Unavailable
	ErrUserNotOnline: 503,
}

// GetHTTPStatus 获取错误对应的HTTP状态码
func GetHTTPStatus(err error) int {
	if status, ok := HTTPStatus[err]; ok {
		return status
	}
	return 500 // 默认内部服务器错误
}
