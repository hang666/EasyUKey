package consts

// 认证状态常量
const (
	AuthStatusPending           = "pending"            // 等待中
	AuthStatusProcessing        = "processing"         // 处理中
	AuthStatusProcessingOnceKey = "processing_oncekey" // 处理OnceKey中
	AuthStatusCompleted         = "completed"          // 已完成
	AuthStatusFailed            = "failed"             // 失败
	AuthStatusExpired           = "expired"            // 已过期
	AuthStatusRejected          = "rejected"           // 被拒绝
)

// 认证结果常量
const (
	AuthResultSuccess = "success" // 成功
	AuthResultFailure = "failure" // 失败
)
