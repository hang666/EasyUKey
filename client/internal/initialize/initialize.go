package initialize

import (
	"github.com/hang666/EasyUKey/client/internal/config"
	"github.com/hang666/EasyUKey/client/internal/global"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// InitAll 初始化所有组件
func InitAll(encryptKeyStr, serverAddr, logLevel, logFile, logConsole, devMode string) error {
	// 初始化配置
	cfg, err := config.InitConfig(encryptKeyStr, serverAddr, logLevel, logFile, logConsole, devMode)
	if err != nil {
		return err
	}
	global.Config = cfg

	// 初始化日志
	_, err = logger.InitClientLogger(cfg.LogLevel, cfg.LogFile, cfg.LogConsole)
	if err != nil {
		return err
	}

	logger.Logger.Info(
		"客户端初始化完成",
		"version", cfg.Version,
		"server_addr", cfg.ServerAddr,
		"log_level", cfg.LogLevel,
		"dev_mode", devMode,
	)

	return nil
}
