package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/shared/pkg/errors"
)

// AdminPanel 管理员面板页面
func AdminPanel(c echo.Context) error {
	return c.Render(http.StatusOK, "admin.html", nil)
}

// VerifyAdminKey 验证管理员密钥
func VerifyAdminKey(c echo.Context) error {
	var req struct {
		AdminKey string `json:"admin_key"`
	}

	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.AdminKey == "" {
		return errors.ErrMissingAdminKey
	}

	// 验证管理员密钥
	apiKey, err := service.GetAPIKey(req.AdminKey)
	if err != nil {
		return errors.ErrInvalidKey
	}
	if !apiKey.IsAdmin {
		return errors.ErrInvalidKey
	}

	result := map[string]interface{}{
		"valid": true,
		"role":  "admin",
	}

	return c.JSON(http.StatusOK, &response.Response{Success: true, Message: "管理员身份验证成功", Data: &result})
}
