package initialize

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"github.com/hang666/EasyUKey/server/internal/config"
	"github.com/hang666/EasyUKey/server/internal/global"
	"github.com/hang666/EasyUKey/server/internal/model/entity"
	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// generateRandomAPIKey 生成随机API密钥
func generateRandomAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// InitDatabase 初始化数据库连接
func InitDatabase(cfg *config.DatabaseConfig) error {
	dsn := global.Config.GetDatabaseDSN()

	// 配置GORM日志
	var gormLogLevel gormLogger.LogLevel
	switch global.Config.Log.Level {
	case "debug":
		gormLogLevel = gormLogger.Info
	case "info":
		gormLogLevel = gormLogger.Warn
	default:
		gormLogLevel = gormLogger.Error
	}

	gormConfig := &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogLevel),
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(cfg.ConnectionMaxLifetime)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	global.DB = db
	logger.Logger.Info("数据库连接成功", "host", cfg.Host, "database", cfg.Database)

	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	if global.DB == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 定义需要迁移的实体
	entities := []interface{}{
		&entity.User{},
		&entity.Device{},
		&entity.AuthSession{},
		&entity.APIKey{},
	}

	// 执行自动迁移
	for _, ent := range entities {
		if err := global.DB.AutoMigrate(ent); err != nil {
			return fmt.Errorf("迁移表结构失败 %T: %w", ent, err)
		}
	}

	logger.Logger.Info("数据库表结构迁移完成")
	return nil
}

// CreateDefaultData 创建默认数据
func CreateDefaultData() error {
	if global.DB == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 检查是否存在管理员API密钥
	var adminCount int64
	if err := global.DB.Model(&entity.APIKey{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
		return fmt.Errorf("检查管理员API密钥失败: %w", err)
	}

	// 如果没有管理员密钥，自动生成一个
	if adminCount == 0 {
		apiKey, err := generateRandomAPIKey()
		if err != nil {
			return fmt.Errorf("生成随机API密钥失败: %w", err)
		}

		adminAPIKey := entity.APIKey{
			Name:        "admin",
			APIKey:      apiKey,
			Description: "系统自动生成的管理员API密钥",
			IsActive:    true,
			IsAdmin:     true,
		}

		if err := global.DB.Create(&adminAPIKey).Error; err != nil {
			return fmt.Errorf("创建管理员API密钥失败: %w", err)
		}

		// 在命令行输出管理员密钥
		fmt.Printf("🔑 系统已自动生成管理员API密钥，请妥善保存：\n")
		fmt.Printf("📋 API Key: %s\n", apiKey)
		fmt.Printf("💡 使用说明：\n")
		fmt.Printf("   - 此密钥具有管理员权限，可以访问所有管理接口\n")
		fmt.Printf("   - 请立即保存此密钥，系统不会再次显示\n")
		fmt.Printf("   - 建议部署完成后创建新的管理员密钥并删除此密钥\n")
		fmt.Printf("   - 可通过管理接口 /api/v1/admin/apikeys 管理API密钥\n")
	}

	return nil
}
