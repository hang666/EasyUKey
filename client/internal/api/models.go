package api

// ConfirmActionPayload /confirm接口的请求体
type ConfirmActionPayload struct {
	Action  string `json:"action"`
	Request string `json:"request"`
	PIN     string `json:"pin,omitempty"` // 添加PIN字段
}

// ConfirmActionResponse 确认操作的响应
type ConfirmActionResponse struct {
	Message       string              `json:"message"`
	Status        ConfirmActionStatus `json:"status"`
	ConfirmStatus bool                `json:"confirmStatus"`
}

type ConfirmActionStatus string

const (
	ConfirmActionStatusSuccess ConfirmActionStatus = "success"
	ConfirmActionStatusError   ConfirmActionStatus = "error"
)

// PINSetupPayload PIN设置的请求体
type PINSetupPayload struct {
	PIN string `json:"pin"`
}

// PINSetupResponse PIN设置的响应
type PINSetupResponse struct {
	Message string              `json:"message"`
	Status  ConfirmActionStatus `json:"status"`
}
