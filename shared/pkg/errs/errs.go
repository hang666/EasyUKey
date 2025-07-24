package errs

import "errors"

// 常用错误定义
var (
	// WebSocket错误
	ErrWSConnectionClosed = errors.New("WebSocket连接已关闭")
	ErrWSTimeout          = errors.New("网络请求超时")
	ErrWSChannelFull      = errors.New("发送通道已满")
	ErrWSNotConnected     = errors.New("WebSocket未连接")
	ErrWSConnectFailed    = errors.New("WebSocket连接失败")

	// 认证错误
	ErrUserRejected = errors.New("用户拒绝认证")

	// 设备错误
	ErrDeviceNotActive     = errors.New("设备未激活")
	ErrDeviceNotFound      = errors.New("设备不存在")
	ErrDeviceNotAvailable  = errors.New("设备信息不可用")
	ErrDeviceAlreadyExists = errors.New("设备已存在")
	ErrDeviceAlreadyBound  = errors.New("设备已绑定用户")

	// 设备组错误
	ErrDeviceGroupNotFound    = errors.New("设备组不存在")
	ErrDeviceGroupNotActive   = errors.New("设备组未激活")
	ErrDeviceGroupNameEmpty   = errors.New("设备组名称不能为空")
	ErrDeviceGroupPermissions = errors.New("设备组权限格式错误")

	// 用户错误
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserAlreadyExists = errors.New("用户名已存在")
	ErrUserNotOnline     = errors.New("用户未在线")

	// 会话错误
	ErrSessionNotFound  = errors.New("认证会话不存在")
	ErrSessionExpired   = errors.New("认证会话已过期")
	ErrSessionCompleted = errors.New("认证会话已完成")

	// 消息错误
	ErrMessageEmpty         = errors.New("消息不能为空")
	ErrMessageTypeEmpty     = errors.New("消息类型不能为空")
	ErrMessageDataEmpty     = errors.New("消息数据不能为空")
	ErrSerializationFailed  = errors.New("序列化失败")
	ErrWSParse              = errors.New("消息解析失败")
	ErrWSValidation         = errors.New("消息验证失败")
	ErrDeviceNotFoundClient = errors.New("设备未找到")

	// 参数错误
	ErrMissingAPIKey     = errors.New("缺少API密钥")
	ErrMissingSessionID  = errors.New("缺少会话ID")
	ErrMissingChallenge  = errors.New("缺少挑战码")
	ErrMissingUsername   = errors.New("缺少用户名")
	ErrMissingName       = errors.New("缺少名称")
	ErrMissingDeviceInfo = errors.New("缺少设备标识信息")
	ErrMissingAdminKey   = errors.New("缺少管理员密钥")
	ErrAPIKeyInvalid     = errors.New("无效的API密钥")
	ErrInvalidRequest    = errors.New("请求参数格式错误")
	ErrInvalidKey        = errors.New("错误的密钥")
	ErrInvalidDeviceID   = errors.New("设备ID格式错误")
	ErrPermissionDenied  = errors.New("权限不足")

	// 身份管理错误
	ErrKeyTooShort           = errors.New("密钥长度不足")
	ErrCipherTextTooShort    = errors.New("密文长度过短")
	ErrCipherBlockSize       = errors.New("密文长度不是块大小的倍数")
	ErrDataEmpty             = errors.New("数据为空")
	ErrInvalidPaddingSize    = errors.New("无效的填充大小")
	ErrInvalidPaddingContent = errors.New("无效的填充内容")
	ErrSharedKeyNotComputed  = errors.New("共享密钥未计算")
	ErrPINOrKeyEmpty         = errors.New("PIN或密钥为空")

	// 回调错误
	ErrCallbackSessionIDMissing = errors.New("session_id is required")
	ErrCallbackUserIDMissing    = errors.New("user_id is required")
	ErrCallbackStatusMissing    = errors.New("status is required")
	ErrCallbackChallengeMissing = errors.New("challenge is required")
	ErrCallbackTimestampMissing = errors.New("timestamp is required")
	ErrCallbackSignatureMissing = errors.New("signature is required")
	ErrCallbackInvalidSignature = errors.New("invalid signature")

	// 其他常用错误
	ErrConvertWSURLFailed = errors.New("转换WebSocket URL失败")
	ErrKeyExchangeFailed  = errors.New("密钥协商失败")
	ErrKeyExchangeTimeout = errors.New("密钥协商超时")
	ErrShowPageFailed     = errors.New("无法显示认证页面")
	ErrWaitConfirmFailed  = errors.New("等待用户确认失败")
)
