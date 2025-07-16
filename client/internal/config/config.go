package config

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/hang666/EasyUKey/client/utils/uid"
)

// Config 客户端配置结构
type Config struct {
	// 服务器配置
	ServerAddr string

	// 安全配置
	EncryptKey    []byte
	EncryptKeyStr string

	// 日志配置
	LogLevel   string
	LogFile    string
	LogConsole string

	// 应用配置
	Version  string
	HTTPPort int
	ExeDir   string
	DevMode  string
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// 应用常量
const (
	ClientVersion = "0.0.1"
	HttpPort      = 18765
)

// InitConfig 初始化配置
func InitConfig(encryptKeyStr, serverAddr, logLevel, logFile, logConsole, devMode string) (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exeDir := filepath.Dir(exePath)

	// 设置默认值
	if encryptKeyStr == "" {
		encryptKeyStr = "123456"
	}
	if serverAddr == "" {
		serverAddr = "http://127.0.0.1:8888"
	}
	if logLevel == "" {
		logLevel = "info"
	}
	if logFile == "" {
		logFile = "logs/client.log"
	}
	if logConsole == "" {
		logConsole = "true"
	}
	if devMode == "" {
		devMode = "false"
	}
	uid.DevMode = devMode

	// 生成加密密钥
	hash := md5.New()
	hash.Write([]byte(encryptKeyStr))
	encryptKey := []byte(hex.EncodeToString(hash.Sum(nil)))

	GlobalConfig = &Config{
		ServerAddr:    serverAddr,
		EncryptKey:    encryptKey,
		EncryptKeyStr: encryptKeyStr,
		LogLevel:      logLevel,
		LogFile:       logFile,
		LogConsole:    logConsole,
		Version:       ClientVersion,
		HTTPPort:      HttpPort,
		ExeDir:        exeDir,
		DevMode:       devMode,
	}

	return GlobalConfig, nil
}
