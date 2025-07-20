package main

import (
	"context"
	"embed"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"

	"github.com/hang666/EasyUKey/server/internal/api"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/initialize"
	"github.com/hang666/EasyUKey/server/internal/middleware"
	"github.com/hang666/EasyUKey/server/internal/router"
	"github.com/hang666/EasyUKey/server/internal/service"
	"github.com/hang666/EasyUKey/server/internal/ws"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

//go:embed template
var TemplateFS embed.FS

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.Parse()

	if err := initialize.InitAll(configPath); err != nil {
		panic("服务器初始化失败: " + err.Error())
	}

	e := echo.New()

	templateRenderer := api.NewEmbedTemplateRenderer(TemplateFS)
	e.Renderer = templateRenderer

	e.HTTPErrorHandler = middleware.ErrorHandler

	middleware.SetupMiddleware(e)
	router.SetupRoutes(e)

	// 初始化WebSocket Hub
	wsHub := ws.NewHub()
	service.SetWSHub(wsHub)

	ws.InitUpgrader()
	ws.GlobalStatusSync.InitWithConfig()

	go wsHub.Run()

	serverAddr := global.Config.GetServerAddr()
	logger.Logger.Info("正在启动EasyUKey认证服务器", "address", serverAddr)

	go func() {
		if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			logger.Logger.Error("服务器启动失败", "error", err)
			os.Exit(1)
		}
	}()

	logger.Logger.Info("EasyUKey认证服务器已启动", "address", serverAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), global.Config.Server.GracefulShutdown)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Logger.Error("服务器关闭失败", "error", err)
	}

	if global.DB != nil {
		sqlDB, err := global.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}

	logger.Logger.Info("EasyUKey认证服务器已关闭")
}
