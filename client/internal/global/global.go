package global

import (
	"github.com/hang666/EasyUKey/client/internal/config"
	"github.com/hang666/EasyUKey/client/internal/pin"
)

// Config 全局配置
var Config *config.Config

// PinManager 全局PIN管理器
var PinManager *pin.PINManager

// SecureStoragePath 全局安全存储路径
var SecureStoragePath string
