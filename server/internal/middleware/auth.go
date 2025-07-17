package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
)

// APIAuth 统一API身份验证中间件
func APIAuth(requireAdmin bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 从请求头获取API密钥
			apiKey := c.Request().Header.Get("X-API-Key")
			if apiKey == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "MISSING_API_KEY",
						"message": "缺少API密钥",
					},
				})
			}

			// 验证API密钥
			var key entity.APIKey
			query := global.DB.Where("api_key = ? AND is_active = ?", apiKey, true)

			if requireAdmin {
				query = query.Where("is_admin = ?", true)
			}

			if err := query.First(&key).Error; err != nil {
				var message string
				if requireAdmin {
					message = "无效的管理员密钥"
				} else {
					message = "无效的API密钥"
				}
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "INVALID_API_KEY",
						"message": message,
					},
				})
			}

			// 如果需要管理员权限但API密钥不是管理员密钥
			if requireAdmin && !key.IsAdmin {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "INSUFFICIENT_PERMISSIONS",
						"message": "需要管理员权限",
					},
				})
			}

			// 将API密钥信息存储在上下文中
			c.Set("api_key", &key)

			return next(c)
		}
	}
}

// AdminAuth 管理员身份验证中间件（保持向后兼容）
func AdminAuth() echo.MiddlewareFunc {
	return APIAuth(true)
}
