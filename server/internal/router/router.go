package router

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/sdk/response"
	"github.com/hang666/EasyUKey/server/internal/api"
	"github.com/hang666/EasyUKey/server/internal/middleware"
	"github.com/hang666/EasyUKey/server/internal/ws"
)

// SetupRoutes 设置路由
func SetupRoutes(e *echo.Echo) {
	// 健康检查
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, &response.Response{
			Success: true,
			Message: "服务正常",
		})
	})

	// WebSocket连接
	e.GET("/ws", ws.HandleWebSocket)

	// 管理员面板页面（无需认证）
	e.GET("/admin", api.AdminPanel)

	// API路由组
	apiV1 := e.Group("/api/v1")

	// 认证相关路由（需要普通API密钥）
	auth := apiV1.Group("", middleware.APIAuth(false))
	{
		auth.POST("/auth", api.StartAuth)
		auth.POST("/auth/verify", api.VerifyAuth)
	}

	// 管理员验证路由（无需认证）
	apiV1.POST("/admin/verify", api.VerifyAdminKey)

	// 管理员路由组（需要管理员API密钥）
	admin := apiV1.Group("/admin", middleware.APIAuth(true))
	{
		// 用户管理
		admin.POST("/users", api.CreateUser)
		admin.GET("/users", api.GetUsers)
		admin.GET("/users/:id", api.GetUser)
		admin.PUT("/users/:id", api.UpdateUser)
		admin.DELETE("/users/:id", api.DeleteUser)
		admin.GET("/users/:username/devices", api.GetUserDevices)

		// 设备管理
		admin.GET("/devices", api.GetDevices)
		admin.GET("/devices/statistics", api.GetDeviceStatistics)
		admin.GET("/devices/pending-activation", api.GetPendingActivationDevices)
		admin.GET("/devices/:id", api.GetDevice)
		admin.PUT("/devices/:id", api.UpdateDevice)
		admin.DELETE("/devices/:id", api.DeleteDevice)
		admin.POST("/devices/:id/offline", api.OfflineDevice)

		// 设备组管理
		admin.GET("/device-groups", api.GetDeviceGroups)
		admin.GET("/device-groups/:id", api.GetDeviceGroup)
		admin.PUT("/device-groups/:id", api.UpdateDeviceGroup)
		admin.PUT("/device-groups/:id/user", api.LinkDeviceGroupUser)

		// API密钥管理
		admin.POST("/apikeys", api.CreateAPIKey)
		admin.GET("/apikeys", api.GetAPIKeys)
		admin.DELETE("/apikeys/:id", api.DeleteAPIKey)

		// 认证会话管理
		admin.GET("/sessions", api.GetAuthSessions)
	}
}
