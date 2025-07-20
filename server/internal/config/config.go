package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Security  SecurityConfig  `mapstructure:"security"`
	Log       LogConfig       `mapstructure:"log"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	HTTP      HTTPConfig      `mapstructure:"http"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host             string        `mapstructure:"host"`
	Port             int           `mapstructure:"port"`
	GracefulShutdown time.Duration `mapstructure:"graceful_shutdown"` // 优雅关闭超时
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host                  string        `mapstructure:"host"`
	Port                  int           `mapstructure:"port"`
	Username              string        `mapstructure:"username"`
	Password              string        `mapstructure:"password"`
	Database              string        `mapstructure:"database"`
	Charset               string        `mapstructure:"charset"`
	MaxIdleConnections    int           `mapstructure:"max_idle_connections"`
	MaxOpenConnections    int           `mapstructure:"max_open_connections"`
	ConnectionMaxLifetime time.Duration `mapstructure:"connection_max_lifetime"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EncryptionKey string `mapstructure:"encryption_key"` // 数据加密密钥
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	// 连接超时配置
	WriteWait  time.Duration `mapstructure:"write_wait"`  // 写入超时时间
	PongWait   time.Duration `mapstructure:"pong_wait"`   // pong等待时间
	PingPeriod time.Duration `mapstructure:"ping_period"` // ping发送周期

	// 消息配置
	MaxMessageSize int64 `mapstructure:"max_message_size"` // 最大消息大小

	// 缓冲区配置
	SendChannelBuffer int `mapstructure:"send_channel_buffer"` // 发送通道缓冲区大小
	ReadBufferSize    int `mapstructure:"read_buffer_size"`    // 读取缓冲区大小
	WriteBufferSize   int `mapstructure:"write_buffer_size"`   // 写入缓冲区大小

	// 压缩配置
	EnableCompression bool `mapstructure:"enable_compression"` // 是否启用压缩

	// 连接限制
	MaxConnections    int           `mapstructure:"max_connections"`    // 最大连接数
	ConnectionTimeout time.Duration `mapstructure:"connection_timeout"` // 连接超时
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"` // 心跳间隔
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	RequestTimeout  time.Duration `mapstructure:"request_timeout"`   // HTTP请求超时
	RateLimit       int           `mapstructure:"rate_limit"`        // 每秒请求限制
	RequestBodySize string        `mapstructure:"request_body_size"` // 请求体大小限制
}

var GlobalConfig *Config

// InitConfig 初始化配置
func InitConfig(configPath string) error {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认查找配置文件
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")

		// 获取可执行文件所在目录
		if execPath, err := os.Executable(); err == nil {
			v.AddConfigPath(filepath.Dir(execPath))
		}
	}

	// 设置环境变量前缀
	v.SetEnvPrefix("EASYUKEY")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Warn("配置文件未找到，使用默认配置和环境变量")
		} else {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
	} else {
		slog.Info("已加载配置文件", "file", v.ConfigFileUsed())
	}

	// 解析配置到结构体
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	GlobalConfig = config
	return nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8888)
	v.SetDefault("server.graceful_shutdown", "30s")

	// 数据库默认配置
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.username", "easyukey")
	v.SetDefault("database.database", "easyukey")
	v.SetDefault("database.charset", "utf8mb4")
	v.SetDefault("database.max_idle_connections", 10)
	v.SetDefault("database.max_open_connections", 100)
	v.SetDefault("database.connection_max_lifetime", "1h")

	// HTTP默认配置
	v.SetDefault("http.request_timeout", "30s")
	v.SetDefault("http.rate_limit", 20)
	v.SetDefault("http.request_body_size", "1M")

	// WebSocket默认配置
	v.SetDefault("websocket.write_wait", "10s")
	v.SetDefault("websocket.pong_wait", "60s")
	v.SetDefault("websocket.ping_period", "54s")
	v.SetDefault("websocket.max_message_size", 512)
	v.SetDefault("websocket.send_channel_buffer", 256)
	v.SetDefault("websocket.read_buffer_size", 1024)
	v.SetDefault("websocket.write_buffer_size", 1024)
	v.SetDefault("websocket.enable_compression", false)
	v.SetDefault("websocket.max_connections", 1000)
	v.SetDefault("websocket.connection_timeout", "30s")
	v.SetDefault("websocket.heartbeat_interval", "30s")

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
}

// GetDatabaseDSN 获取数据库连接字符串
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.Charset)
}

// GetServerAddr 获取服务器地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetBatchSize 获取批处理大小（基于WebSocket连接数动态计算）
func (c *Config) GetBatchSize() int {
	return c.WebSocket.MaxConnections / 20 // 连接数的5%
}

// GetSyncInterval 获取同步间隔（基于心跳间隔动态计算）
func (c *Config) GetSyncInterval() time.Duration {
	return time.Duration(c.WebSocket.HeartbeatInterval.Nanoseconds() / 6) // 心跳间隔的1/6
}

// GetUpdateChannelBuffer 获取更新通道缓冲区大小（基于WebSocket连接数动态计算）
func (c *Config) GetUpdateChannelBuffer() int {
	return c.WebSocket.MaxConnections / 10 // 连接数的10%
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务器端口必须在1-65535范围内")
	}
	if c.Server.GracefulShutdown <= 0 {
		return fmt.Errorf("优雅关闭超时时间必须大于0")
	}

	// 验证数据库配置
	if c.Database.Host == "" {
		return fmt.Errorf("数据库主机不能为空")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("数据库端口必须在1-65535范围内")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("数据库用户名不能为空")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("数据库名不能为空")
	}
	if c.Database.MaxIdleConnections < 0 {
		return fmt.Errorf("数据库最大空闲连接数不能为负数")
	}
	if c.Database.MaxOpenConnections < 0 {
		return fmt.Errorf("数据库最大打开连接数不能为负数")
	}
	if c.Database.MaxIdleConnections > c.Database.MaxOpenConnections && c.Database.MaxOpenConnections > 0 {
		return fmt.Errorf("数据库最大空闲连接数不能大于最大打开连接数")
	}
	if c.Database.ConnectionMaxLifetime < 0 {
		return fmt.Errorf("数据库连接最大生存时间不能为负数")
	}

	// 验证安全配置
	if c.Security.EncryptionKey == "" {
		return fmt.Errorf("加密密钥不能为空")
	}

	// 验证HTTP配置
	if c.HTTP.RequestTimeout <= 0 {
		return fmt.Errorf("HTTP请求超时时间必须大于0")
	}
	if c.HTTP.RateLimit <= 0 {
		return fmt.Errorf("速率限制必须大于0")
	}
	if c.HTTP.RequestBodySize == "" {
		return fmt.Errorf("请求体大小限制不能为空")
	}

	// 验证WebSocket配置
	if c.WebSocket.WriteWait <= 0 {
		return fmt.Errorf("WebSocket写入超时时间必须大于0")
	}
	if c.WebSocket.PongWait <= 0 {
		return fmt.Errorf("WebSocket pong等待时间必须大于0")
	}
	if c.WebSocket.PingPeriod <= 0 {
		return fmt.Errorf("WebSocket ping周期必须大于0")
	}
	if c.WebSocket.PingPeriod >= c.WebSocket.PongWait {
		return fmt.Errorf("WebSocket ping周期必须小于pong等待时间")
	}
	if c.WebSocket.MaxMessageSize <= 0 {
		return fmt.Errorf("WebSocket最大消息大小必须大于0")
	}
	if c.WebSocket.SendChannelBuffer <= 0 {
		return fmt.Errorf("WebSocket发送通道缓冲区大小必须大于0")
	}
	if c.WebSocket.ReadBufferSize <= 0 {
		return fmt.Errorf("WebSocket读取缓冲区大小必须大于0")
	}
	if c.WebSocket.WriteBufferSize <= 0 {
		return fmt.Errorf("WebSocket写入缓冲区大小必须大于0")
	}
	if c.WebSocket.MaxConnections <= 0 {
		return fmt.Errorf("WebSocket最大连接数必须大于0")
	}
	if c.WebSocket.ConnectionTimeout <= 0 {
		return fmt.Errorf("WebSocket连接超时时间必须大于0")
	}
	if c.WebSocket.HeartbeatInterval <= 0 {
		return fmt.Errorf("WebSocket心跳间隔必须大于0")
	}

	return nil
}
