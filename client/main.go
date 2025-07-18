package main

import (
	"embed"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hang666/EasyUKey/client/internal/api"
	"github.com/hang666/EasyUKey/client/internal/confirmation"
	"github.com/hang666/EasyUKey/client/internal/device"
	"github.com/hang666/EasyUKey/client/internal/global"
	"github.com/hang666/EasyUKey/client/internal/initialize"
	"github.com/hang666/EasyUKey/client/internal/pin"
	"github.com/hang666/EasyUKey/client/internal/ws"
	"github.com/hang666/EasyUKey/shared/pkg/identity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

//go:embed template
var TemplateFS embed.FS

// 编译时注入的配置变量
var (
	EncryptKeyStr string
	ServerAddr    string
	LogLevel      string
	LogFile       string
	LogConsole    string
	DevMode       string
)

func main() {
	if err := initialize.InitAll(EncryptKeyStr, ServerAddr, LogLevel, LogFile, LogConsole, DevMode); err != nil {
		panic("客户端初始化失败: " + err.Error())
	}

	go device.StartTimer(global.Config.ExeDir)

	for device.DeviceInfo.GetDevice() == nil {
		time.Sleep(1 * time.Second)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	startServices()

	logger.Logger.Info("EasyUKey Client 已启动")

	<-sigChan
	shutdown()
}

// startServices 启动各个服务
func startServices() {
	// 设置全局安全存储路径
	global.SecureStoragePath = filepath.Join(global.Config.ExeDir, ".secure")

	isInitialized := identity.IsInitialized(global.SecureStoragePath)
	if !isInitialized {
		logger.Logger.Info("设备未初始化，将在连接后进行初始化")
	}

	// 初始化PIN管理器
	global.PinManager = pin.NewPINManager()

	confirmation.Init(global.Config.HTTPPort)
	ws.Init(global.Config.ServerAddr, isInitialized)

	go func() {
		if err := api.StartHttpServer(global.Config.HTTPPort, TemplateFS); err != nil {
			logger.Logger.Error("HTTP服务器启动失败", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		if err := ws.Connect(); err != nil {
			logger.Logger.Error("WebSocket连接失败", "error", err)
			os.Exit(1)
		}
	}()

	go ws.MonitorConnection()
}

// shutdown 优雅关闭服务
func shutdown() {
	shutdownStart := time.Now()
	logger.Logger.Info("收到关闭信号，正在优雅关闭...")

	device.StopTimer()
	api.StopHttpServer()
	ws.Disconnect()

	// 关闭PIN管理器
	if global.PinManager != nil {
		global.PinManager.Close()
	}

	logger.Logger.Info("EasyUKey Client 关闭完成", "耗时", time.Since(shutdownStart))
}
