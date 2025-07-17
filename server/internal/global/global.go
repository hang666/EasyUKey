package global

import (
	"gorm.io/gorm"

	"github.com/hang666/EasyUKey/server/internal/config"
)

var (
	// DB 全局数据库连接
	DB *gorm.DB

	// Config 全局配置
	Config *config.Config
)
