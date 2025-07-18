package errs

import "errors"

// 核心错误定义
var (
	// API错误
	ErrAPIKeyInvalid    = errors.New("无效的API密钥")
	ErrAPIError         = errors.New("API调用错误")
	ErrInvalidRequest   = errors.New("请求参数错误")
	ErrPermissionDenied = errors.New("权限不足")

	// 数据错误
	ErrSerializationFailed   = errors.New("序列化失败")
	ErrRequestCreationFailed = errors.New("创建请求失败")
	ErrRequestFailed         = errors.New("请求失败")
	ErrResponseParseFailed   = errors.New("解析响应失败")
	ErrDataParseFailed       = errors.New("解析数据失败")
	ErrInvalidResponseFormat = errors.New("响应格式错误")

	// 认证错误
	ErrAuthTimeout               = errors.New("认证超时")
	ErrAuthStartFailed           = errors.New("发起认证失败")
	ErrRandomGenerationFailed    = errors.New("生成随机数失败")
	ErrChallengeGenerationFailed = errors.New("生成挑战码失败")

	// 回调错误
	ErrInvalidJSON              = errors.New("JSON格式错误")
	ErrMissingSessionID         = errors.New("缺少会话ID")
	ErrMissingUserID            = errors.New("缺少用户ID")
	ErrMissingStatus            = errors.New("缺少状态")
	ErrMissingChallenge         = errors.New("缺少挑战码")
	ErrMissingTimestamp         = errors.New("缺少时间戳")
	ErrMissingSignature         = errors.New("缺少签名")
	ErrTimestampOutOfRange      = errors.New("时间戳超出范围")
	ErrInvalidSignature         = errors.New("签名无效")
	ErrCallbackValidationFailed = errors.New("回调验证失败")
	ErrUnknownCallbackStatus    = errors.New("未知回调状态")
)

// HTTPStatusMap 定义错误对应的HTTP状态码
var HTTPStatusMap = map[error]int{
	ErrInvalidRequest:   400,
	ErrAPIKeyInvalid:    401,
	ErrPermissionDenied: 403,
}

// GetHTTPStatus 获取错误对应的HTTP状态码
func GetHTTPStatus(err error) int {
	if status, ok := HTTPStatusMap[err]; ok {
		return status
	}
	return 500
}
