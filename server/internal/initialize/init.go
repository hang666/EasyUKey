package initialize

import (
	"fmt"

	"github.com/hang666/EasyUKey/server/internal/config"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// InitAll 初始化所有组件
func InitAll(configPath string) error {
	// 1. 初始化配置
	if err := config.InitConfig(configPath); err != nil {
		return fmt.Errorf("配置初始化失败: %w", err)
	}
	global.Config = config.GlobalConfig

	// 2. 初始化日志系统
	loggerConfig := &logger.LogConfig{
		Level:   global.Config.Log.Level,
		Format:  global.Config.Log.Format,
		Output:  global.Config.Log.Output,
		Console: "true", // 默认启用控制台输出
	}
	log, err := logger.InitLogger(loggerConfig)
	if err != nil {
		return fmt.Errorf("日志系统初始化失败: %w", err)
	}
	_ = log // 日志实例已设置为全局变量

	// 3. 初始化数据库连接
	if err := InitDatabase(&global.Config.Database); err != nil {
		return fmt.Errorf("数据库初始化失败: %w", err)
	}

	// 4. 自动迁移数据库表结构
	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 5. 创建默认数据
	if err := CreateDefaultData(); err != nil {
		return fmt.Errorf("创建默认数据失败: %w", err)
	}

	logger.Logger.Info("服务器初始化完成")
	return nil
}
