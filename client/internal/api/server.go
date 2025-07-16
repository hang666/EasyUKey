package api

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"time"

	"github.com/hang666/EasyUKey/shared/pkg/logger"

	"github.com/labstack/echo/v4"
)

var httpServer *echo.Echo

// StartHttpServer 启动HTTP服务器
func StartHttpServer(port int, templateFS embed.FS) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	subFS, err := fs.Sub(templateFS, "template")
	if err != nil {
		logger.Logger.Error("无法找到嵌入的template目录", "error", err)
		os.Exit(1)
	}

	// 注册模板渲染器
	t := &TemplateRenderer{
		Templates: template.Must(template.New("").ParseFS(subFS, "*.html")),
	}
	e.Renderer = t

	// 认证相关路由
	e.GET("/", HandleConfirmPage)
	e.POST("/confirm", HandleConfirmAction)

	// PIN设置相关路由
	e.GET("/pin", HandlePINPage)
	e.POST("/pin-setup", HandlePINSetup)

	httpServer = e

	logger.Logger.Info("HTTP服务器启动", "port", port)
	return httpServer.Start(fmt.Sprintf("localhost:%d", port))
}

// StopHttpServer 停止HTTP服务器
func StopHttpServer() {
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
		logger.Logger.Info("HTTP服务器已停止")
	}
}
