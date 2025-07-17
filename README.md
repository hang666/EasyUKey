# EasyUKey

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Development Status](https://img.shields.io/badge/status-Development-orange)](https://github.com/hang666/EasyUKey)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/hang666/EasyUKey)

EasyUKey 是一个基于USB设备的企业级身份认证解决方案，提供高安全性的硬件身份验证服务。采用现代化的客户端-服务器架构，集成实时WebSocket通信、多重加密保护和TOTP双因子认证，为企业应用提供简单易用的硬件认证集成方案。

> ⚠️ **开发状态**：本项目目前处于活跃开发阶段，核心功能已稳定实现，安全特性持续优化中。建议在测试环境充分验证后再用于生产环境。

## 📖 目录

- [系统架构](#-系统架构)
- [核心特性](#-核心特性)
- [快速开始](#-快速开始)
- [安装部署](#-安装部署)
- [配置指南](#-配置指南)
- [API文档](#-api文档)
- [SDK使用](#-sdk使用)
- [认证流程](#-认证流程)
- [使用场景](#-使用场景)
- [故障排除](#-故障排除)
- [开发指南](#-开发指南)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   第三方应用     │    │   EasyUKey      │    │   USB客户端     │
│                │    │   服务器        │    │                │
│  ┌──────────────┤    ├──────────────┐  │    │  ┌──────────────┤
│  │   SDK集成    │◄──►│  RESTful API │  │    │  │   认证客户端  │
│  └──────────────│    │              │  │    │  └──────────────│
│  ┌──────────────┤    │  ┌───────────┤  │    │  ┌──────────────┤
│  │   异步回调    │◄──►│  │ WebSocket │◄─┼────┼─►│  WebSocket   │
│  └──────────────│    │  │    Hub    │  │    │  │   连接       │
└─────────────────┘    │  └───────────┤  │    │  └──────────────│
                       │  ┌───────────┤  │    │  ┌──────────────┤
                       │  │  数据库   │  │    │  │   硬件检测    │
                       │  │   MySQL   │  │    │  │   U盘识别    │
                       │  └───────────┘  │    │  └──────────────│
                       └─────────────────┘    └─────────────────┘
```

### 架构组件

- **EasyUKey服务器**：核心认证服务，管理用户、设备和认证会话
- **USB客户端**：部署在用户USB设备上的认证客户端程序
- **Go SDK**：为第三方应用提供的集成开发包
- **MySQL数据库**：存储用户、设备、API密钥等核心数据

## 🚀 核心特性

### 🔐 企业级安全防护

- **🔑 硬件身份验证**：基于USB设备的物理身份验证，防止软件层面攻击
- **🎯 双重设备识别**：同时验证U盘分区序列号和设备序列号，确保硬件唯一性
- **🛡️ OnceKey防复制**：动态一次性密钥机制，有效防止硬件复制攻击
- **🔐 多层加密保护**：
  - ECDH密钥交换（P-256椭圆曲线）
  - AES-256-GCM端到端加密通信
  - PIN码 + 加密密钥的安全存储方案
- **📱 TOTP双因子认证**：集成时间基础的一次性密码验证
- **✅ 数字签名验证**：回调请求HMAC-SHA256签名防篡改

### 🌐 高性能通信架构

- **🔄 WebSocket Hub**：高并发连接管理，支持单点登录策略
- **🔒 端到端加密**：所有通信消息端到端加密传输
- **💓 智能心跳检测**：连接状态监控和自动重连机制
- **📡 实时状态同步**：设备在线状态实时同步更新
- **⚡ 异步回调机制**：支持认证结果异步回调通知

### 🛠️ 开发者友好

- **📦 完整Go SDK**：提供认证、设备管理、用户管理等全功能API
- **🌐 RESTful API**：标准化的HTTP API接口设计
- **📋 详细文档**：完整的API文档和使用示例
- **🧪 测试支持**：内置测试用例和开发环境配置

## 🔄 认证流程

### 完整认证流程图

```
第三方应用          EasyUKey服务器         USB客户端
     │                    │                   │
     │  1. 发起认证请求     │                   │
     ├──────────────────► │                   │
     │                    │  2. 转发认证请求   │
     │                    ├──────────────────► │
     │                    │                   │ 3. 硬件验证
     │                    │                   │ - 设备序列号验证
     │                    │                   │ - 卷序列号验证
     │                    │                   │ - PIN码验证
     │                    │                   │
     │                    │  4. 生成认证响应   │
     │                    │ ◄─────────────────┤
     │                    │                   │
     │                    │ 5. OnceKey交换    │
     │                    ├─────────────────► │
     │                    │ ◄─────────────────┤
     │                    │                   │
     │  6. 异步回调通知     │                   │
     │ ◄──────────────────┤                   │
     │                    │                   │
```

### 认证步骤详解

1. **认证请求发起**
   - 第三方应用通过SDK或REST API向EasyUKey服务器发起认证请求
   - 包含用户标识、认证动作、挑战码等信息

2. **请求转发与设备定位**
   - 服务器定位用户绑定的在线设备
   - 通过WebSocket加密通道转发认证请求到客户端

3. **多重硬件验证**
   - **设备序列号验证**：验证USB设备硬件序列号
   - **卷序列号验证**：验证U盘分区序列号
   - **PIN码验证**：用户输入PIN码进行身份确认
   - **TOTP验证**（可选）：时间基础的动态密码验证

4. **认证响应生成**
   - 客户端基于硬件信息和OnceKey生成认证密钥
   - 使用ECDH密钥交换和AES-256-GCM加密响应数据
   - 通过WebSocket返回认证结果

5. **防复制密钥更新**
   - 认证成功后进行新的OnceKey交换
   - 更新设备端和服务器端的防复制密钥
   - 确保每次认证都使用唯一密钥

6. **结果回调通知**
   - 服务器向第三方应用发送异步回调通知
   - 包含认证结果、签名验证等信息
   - 支持HMAC-SHA256签名防篡改验证

## 🔧 快速开始

### 环境要求

- **数据库**：MySQL 5.7+
- **依赖**：Go, Git, Make

### 安装部署

1. **克隆项目**

```bash
git clone https://github.com/hang666/EasyUKey.git
cd EasyUKey
```

2. **构建应用**

```bash
# 构建服务器
make server
# 构建客户端 需要设置加密密钥和服务器地址
make client ENCRYPT_KEY_STR=123456789 SERVER_ADDR=http://localhost:8888
```

3. **配置服务器**

```bash
# 编辑配置文件，设置数据库连接等
cp server/config.example.yaml server/config.yaml
```

4. **运行服务器**

```bash
cd build
./easyukey-server
```

5. **部署客户端**

将客户端复制到USB设备打开即可使用

## ⚙️ 配置指南

### 服务器配置详解

完整的配置文件示例请参考 [`server/config.example.yaml`](server/config.example.yaml)

#### 核心配置

```yaml
# 服务器监听配置
server:
  host: "0.0.0.0"          # 监听地址
  port: 8888               # 监听端口
  graceful_shutdown: "30s" # 优雅关闭超时

# 数据库连接
database:
  host: "localhost"
  port: 3306
  username: "easyukey"
  password: "your_password"
  database: "easyukey"
  charset: "utf8mb4"
  max_idle_connections: 10
  max_open_connections: 100
  connection_max_lifetime: "1h"

# 安全配置
security:
  encryption_key: "your-32-character-encryption-key"

# WebSocket配置
websocket:
  write_wait: "10s"
  pong_wait: "60s"
  ping_period: "30s"
  max_message_size: 8192
  max_connections: 1000
```

#### 性能调优配置

```yaml
# HTTP服务配置
http:
  request_timeout: "30s"
  rate_limit: 20
  request_body_size: "1M"

# 日志配置
log:
  level: "info"      # debug, info, warn, error
  format: "json"     # json, text
  output: "stdout"   # stdout, stderr, 或文件路径
```

### 客户端配置

客户端主要通过编译时参数配置：

```bash
# 标准配置
make client \
  ENCRYPT_KEY_STR=your_encrypt_key \
  SERVER_ADDR=http://your-server:8888 \
  DEV_MODE=false

# 开发模式配置
make client \
  ENCRYPT_KEY_STR=dev_key_123 \
  SERVER_ADDR=http://localhost:8888 \
  DEV_MODE=true
```

## 📚 API文档

### REST API接口

EasyUKey提供完整的RESTful API，支持用户管理、设备管理和认证功能。

#### 认证API

**发起认证**

```http
POST /api/auth/start
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "user_id": "testuser",
  "challenge": "random-challenge-string",
  "action": "login",
  "message": "请确认登录操作",
  "timeout": 600,
  "callback_url": "https://your-app.com/auth/callback"
}
```

**响应示例**

```json
{
  "success": true,
  "data": {
    "session_id": "uuid-session-id",
    "status": "pending",
    "expires_at": "2024-01-01T12:00:00Z"
  }
}
```

#### 设备管理API

**获取设备列表**

```http
GET /api/admin/devices
Authorization: Bearer admin-api-key
```

**绑定设备到用户**

```http
POST /api/admin/devices/{device_id}/bind
Authorization: Bearer admin-api-key
Content-Type: application/json

{
  "user_id": 1,
  "permissions": ["login", "transaction"]
}
```

#### 用户管理API

**创建用户**

```http
POST /api/admin/users
Authorization: Bearer admin-api-key
Content-Type: application/json

{
  "username": "john.doe",
  "permissions": ["login", "transaction"]
}
```

### WebSocket API

客户端通过WebSocket与服务器进行实时通信：

```javascript
// 连接WebSocket
const ws = new WebSocket('ws://localhost:8888/ws');

// 认证消息
const authMessage = {
  type: 'auth',
  device_id: 'device-uuid',
  token: 'device-token'
};

ws.send(JSON.stringify(authMessage));
```

## 🛠️ SDK使用

### Go SDK快速开始

#### 安装SDK

```bash
go mod init your-app
go get github.com/hang666/EasyUKey/sdk
```

#### 基础使用

```go
package main

import (
    "log"
    "github.com/hang666/EasyUKey/sdk"
    "github.com/hang666/EasyUKey/sdk/request"
)

func main() {
    // 创建客户端
    client := sdk.NewClient("http://localhost:8888", "your-api-key")
    
    // 发起认证
    authResult, err := client.StartAuth("testuser", &request.AuthRequest{
        Challenge:   "random-challenge",
        Timeout:     600,
        UserID:      "testuser",
        Action:      "login",
        Message:     "请确认登录操作",
        CallbackURL: "https://your-app.com/callback",
    })
    
    if err != nil {
        log.Fatalf("认证失败: %v", err)
    }
    
    log.Printf("认证会话ID: %s", authResult.Data.SessionID)
}
```

#### 设备管理

```go
// 获取设备列表
devices, err := client.GetDevices()
if err != nil {
    log.Fatalf("获取设备列表失败: %v", err)
}

for _, device := range devices.Data {
    log.Printf("设备: %s (在线: %v)", device.Name, device.IsOnline)
}

// 绑定设备到用户
err = client.BindDeviceToUser(deviceID, userID, []string{"login", "transaction"})
if err != nil {
    log.Fatalf("绑定设备失败: %v", err)
}
```

#### 用户管理

```go
// 创建用户
user, err := client.CreateUser(&request.CreateUserRequest{
    Username:    "john.doe",
    Permissions: []string{"login", "transaction"},
})
if err != nil {
    log.Fatalf("创建用户失败: %v", err)
}

// 获取用户列表
users, err := client.GetUsers()
if err != nil {
    log.Fatalf("获取用户列表失败: %v", err)
}
```

#### 回调处理

```go
import (
    "net/http"
    "github.com/hang666/EasyUKey/sdk"
)

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
    // 验证回调签名
    isValid := sdk.VerifyCallback(r, "your-api-secret")
    if !isValid {
        http.Error(w, "无效的回调签名", http.StatusUnauthorized)
        return
    }
    
    // 处理认证结果
    var callbackData sdk.CallbackData
    json.NewDecoder(r.Body).Decode(&callbackData)
    
    if callbackData.Result == "success" {
        // 认证成功，处理业务逻辑
        log.Printf("用户 %s 认证成功", callbackData.UserID)
    } else {
        // 认证失败
        log.Printf("用户 %s 认证失败: %s", callbackData.UserID, callbackData.Message)
    }
}
```

## 🎯 使用场景

### 企业级应用

#### 🏢 办公系统集成
- **OA系统登录**：替代传统密码登录，提供硬件级别的身份验证
- **ERP系统**：财务、人事等敏感系统的安全访问控制
- **邮箱系统**：企业邮箱的二次验证保护

#### 💰 金融行业应用
- **网银系统**：个人和企业网银的硬件认证
- **支付系统**：大额转账和交易的安全确认
- **投资平台**：证券、基金交易的身份验证

#### 🏥 医疗健康
- **医院信息系统**：医生工作站的安全登录
- **电子病历系统**：患者隐私数据的访问控制
- **药品管理**：处方开具和药品调配的权限验证

#### 🏭 工业控制
- **数据中心**：服务器和网络设备的物理访问控制
- **生产系统**：工业控制系统的操作员身份验证
- **实验室**：精密设备和数据的访问管理

### 开发集成场景

#### 🌐 Web应用集成
```javascript
// 前端集成示例
async function authenticateWithEasyUKey(userId, action) {
    const response = await fetch('/api/auth/start', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer your-api-key'
        },
        body: JSON.stringify({
            user_id: userId,
            action: action,
            challenge: generateChallenge(),
            message: `请确认${action}操作`,
            timeout: 300
        })
    });
    
    const result = await response.json();
    return result.data.session_id;
}
```

#### 🖥️ 桌面应用集成
```go
// 桌面应用集成示例
func authenticateUser(userID, action string) error {
    client := sdk.NewClient("http://your-server:8888", "your-api-key")
    
    authResult, err := client.StartAuth(userID, &request.AuthRequest{
        Challenge: generateChallenge(),
        Action:    action,
        Message:   fmt.Sprintf("请确认%s操作", action),
        Timeout:   300,
    })
    
    if err != nil {
        return err
    }
    
    // 等待认证结果
    return waitForAuthResult(authResult.Data.SessionID)
}
```

## 🚨 故障排除

### 常见问题

#### 服务器启动问题

**问题：服务器启动失败**
```bash
# 检查端口占用
netstat -tlnp | grep :8888

# 检查数据库连接
mysql -h localhost -u easyukey -p

# 查看服务器日志
./easyukey-server --log-level debug
```

**问题：数据库连接失败**
```yaml
# 检查配置文件中的数据库配置
database:
  host: "localhost"
  port: 3306
  username: "easyukey"
  password: "correct_password"
  database: "easyukey"
```

#### 客户端连接问题

**问题：客户端无法连接服务器**
```bash
# 检查网络连通性
ping your-server-ip
telnet your-server-ip 8888

# 检查防火墙设置
sudo ufw status
sudo firewall-cmd --list-ports
```

**问题：USB设备识别失败**
- 确保USB设备有足够的存储空间
- 检查USB设备的文件系统格式（推荐NTFS或FAT32）
- 验证设备是否具有唯一的序列号

#### 认证流程问题

**问题：认证超时**
```go
// 增加认证超时时间
authRequest := &request.AuthRequest{
    Timeout: 600, // 增加到10分钟
    // ... 其他参数
}
```

**问题：OnceKey验证失败**
- 检查客户端和服务器的系统时间是否同步
- 确认加密密钥配置一致
- 重新初始化设备的OnceKey

### 性能优化

#### 服务器优化

```yaml
# 数据库连接池优化
database:
  max_idle_connections: 20
  max_open_connections: 200
  connection_max_lifetime: "2h"

# WebSocket连接优化
websocket:
  max_connections: 2000
  send_channel_buffer: 512
  read_buffer_size: 8192
  write_buffer_size: 8192
```

#### 网络优化

```yaml
# 启用WebSocket压缩
websocket:
  enable_compression: true

# 调整心跳间隔
websocket:
  ping_period: "15s"
  pong_wait: "30s"
```

### 安全建议

#### 生产环境配置

1. **使用HTTPS/WSS**
```yaml
server:
  tls_cert_file: "/path/to/cert.pem"
  tls_key_file: "/path/to/key.pem"
```

2. **配置防火墙**
```bash
# 只允许必要的端口
sudo ufw allow 8888/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

3. **定期更新密钥**
```bash
# 定期轮换API密钥和加密密钥
# 建议每3-6个月更新一次
```

4. **监控和日志**
```yaml
log:
  level: "info"
  output: "/var/log/easyukey/server.log"
  
# 配置日志轮转
```

### 监控和维护

#### 健康检查

```bash
# 服务器健康检查端点
curl http://localhost:8888/health

# 数据库连接检查
curl http://localhost:8888/health/db

# WebSocket连接数检查
curl http://localhost:8888/metrics
```

#### 性能监控

```go
// 监控认证延迟
func monitorAuthLatency() {
    start := time.Now()
    // ... 执行认证
    latency := time.Since(start)
    log.Printf("认证延迟: %v", latency)
}
```

## 🧪 开发指南

### 开发环境搭建

#### 前置条件

```bash
# 安装必要工具
sudo apt update
sudo apt install -y golang-go mysql-server git make

# 设置Go环境
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

#### 快速启动开发环境

```bash
# 1. 克隆项目
git clone https://github.com/hang666/EasyUKey.git
cd EasyUKey

# 2. 初始化数据库
mysql -u root -p << EOF
CREATE DATABASE easyukey_dev CHARACTER SET utf8mb4;
CREATE USER 'dev'@'localhost' IDENTIFIED BY 'dev123';
GRANT ALL PRIVILEGES ON easyukey_dev.* TO 'dev'@'localhost';
FLUSH PRIVILEGES;
EOF

# 3. 配置开发环境
cp server/config.example.yaml server/config.dev.yaml
# 编辑 config.dev.yaml 设置开发数据库

# 4. 启动开发服务器
cd server
go run main.go -config config.dev.yaml

# 5. 编译开发客户端
cd ../
make client ENCRYPT_KEY_STR=dev_key_123 SERVER_ADDR=http://localhost:8888 DEV_MODE=true
```

### 项目结构解析

```
EasyUKey/
├── client/                 # USB客户端
│   ├── internal/          # 内部包
│   │   ├── api/          # API客户端
│   │   ├── device/       # 设备管理
│   │   ├── ws/           # WebSocket通信
│   │   └── pin/          # PIN码管理
│   ├── template/         # UI模板
│   └── main.go          # 入口文件
├── server/                # 服务器端
│   ├── internal/         # 内部包
│   │   ├── api/         # HTTP API处理
│   │   ├── model/       # 数据模型
│   │   ├── service/     # 业务逻辑
│   │   ├── ws/          # WebSocket Hub
│   │   └── middleware/  # 中间件
│   ├── config.example.yaml
│   └── main.go
├── sdk/                  # Go SDK
│   ├── client.go        # SDK客户端
│   ├── admin.go         # 管理功能
│   ├── request/         # 请求结构
│   ├── response/        # 响应结构
│   └── test/           # 测试用例
└── shared/              # 共享包
    └── pkg/
        ├── logger/      # 日志工具
        ├── identity/    # 身份认证
        └── messages/    # 消息定义
```

### 核心模块说明

#### 1. 认证模块 (Authentication)
- **位置**: `server/internal/service/auth.go`
- **功能**: 处理认证请求、会话管理
- **关键接口**: `StartAuth()`, `CompleteAuth()`

#### 2. 设备管理模块 (Device)
- **位置**: `server/internal/service/device.go`
- **功能**: 设备注册、状态管理、权限控制
- **关键接口**: `RegisterDevice()`, `UpdateDeviceStatus()`

#### 3. WebSocket通信模块 (WebSocket)
- **位置**: `server/internal/ws/hub.go`
- **功能**: 实时通信、连接管理
- **关键接口**: `HandleConnection()`, `BroadcastMessage()`

#### 4. 加密模块 (Encryption)
- **位置**: `shared/pkg/identity/`
- **功能**: ECDH密钥交换、AES加密解密
- **关键接口**: `GenerateKeyPair()`, `EncryptMessage()`

### 测试指南

#### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./server/internal/service/
go test ./sdk/test/

# 运行测试并显示覆盖率
go test -cover ./...
```

#### 集成测试

```bash
# 启动测试服务器
cd server
go run main.go -config config.test.yaml &

# 运行SDK集成测试
cd ../sdk/test
go test -v client_test.go

# 运行端到端测试
cd ../../
./scripts/e2e_test.sh
```

#### 性能测试

```bash
# 认证性能测试
go test -bench=BenchmarkAuth ./server/internal/service/

# WebSocket连接性能测试
go test -bench=BenchmarkWebSocket ./server/internal/ws/
```

### 贡献指南

#### 代码规范

1. **Go代码风格**
```bash
# 使用官方格式化工具
go fmt ./...

# 使用静态分析工具
go vet ./...

# 使用linter
golangci-lint run
```

2. **提交信息规范**
```bash
# 功能：feat: 添加用户管理API
# 修复：fix: 修复WebSocket连接泄漏问题
# 文档：docs: 更新API文档
# 样式：style: 统一代码格式
# 重构：refactor: 重构认证模块
# 测试：test: 添加设备管理测试用例
```

#### 开发流程

1. **创建功能分支**
```bash
git checkout -b feature/new-feature
```

2. **开发和测试**
```bash
# 编写代码
# 运行测试
go test ./...
# 确保代码质量
golangci-lint run
```

3. **提交和推送**
```bash
git add .
git commit -m "feat: 添加新功能"
git push origin feature/new-feature
```

4. **创建Pull Request**
- 详细描述功能变更
- 包含必要的测试用例
- 确保CI/CD通过

### API扩展开发

#### 添加新的认证方式

```go
// 1. 在 server/internal/service/auth.go 中添加新方法
func (s *AuthService) StartBiometricAuth(req *request.BiometricAuthRequest) (*response.AuthResponse, error) {
    // 实现生物识别认证逻辑
}

// 2. 在路由中注册新端点
router.POST("/api/auth/biometric", handlers.StartBiometricAuth)

// 3. 在SDK中添加客户端方法
func (c *Client) StartBiometricAuth(req *request.BiometricAuthRequest) (*response.AuthResponse, error) {
    return c.request("POST", "/api/auth/biometric", req)
}
```

#### 添加新的管理功能

```go
// 1. 定义数据模型
type AuditLog struct {
    ID        uint      `json:"id"`
    UserID    uint      `json:"user_id"`
    Action    string    `json:"action"`
    Timestamp time.Time `json:"timestamp"`
}

// 2. 实现服务层
func (s *AdminService) GetAuditLogs(limit, offset int) ([]*AuditLog, error) {
    // 实现审计日志查询
}

// 3. 添加API端点
router.GET("/api/admin/audit-logs", handlers.GetAuditLogs)
```

## 📝 开发路线图

### 当前版本 (v1.0)

- [x] **基础认证功能**：硬件设备认证、PIN码验证
- [x] **通信加密**：ECDH密钥交换、AES-256-GCM加密
- [x] **异步回调**：认证结果异步通知机制
- [x] **OnceKey防复制**：动态一次性密钥机制
- [x] **客户端同步返回**：认证接口同步返回结果

### 下一个版本 (v1.1)

- [ ] **多平台支持**：支持Linux和macOS客户端
- [ ] **生物识别集成**：指纹、面部识别等生物特征认证
- [ ] **移动端支持**：iOS和Android客户端应用
- [ ] **管理界面完善**：Web管理控制台
- [ ] **API网关集成**：支持主流API网关

### 未来版本 (v2.0+)

- [ ] **区块链集成**：去中心化身份验证
- [ ] **零知识证明**：隐私保护的身份验证
- [ ] **多因子认证**：短信、邮箱等多种验证方式
- [ ] **联邦认证**：支持SAML、OAuth2.0等标准协议
- [ ] **AI安全检测**：行为分析和异常检测

### 性能目标

| 指标 | 当前版本 | 目标版本 |
|------|----------|----------|
| 认证延迟 | < 3s | < 1s |
| 并发连接 | 1,000 | 10,000 |
| 吞吐量 | 100 req/s | 1,000 req/s |
| 可用性 | 99.5% | 99.9% |

## 🤝 贡献指南

我们非常欢迎社区贡献！无论是代码贡献、问题反馈还是文档改进，都是对项目的重要支持。

### 🚀 快速贡献

#### 报告问题
1. 在 [GitHub Issues](https://github.com/hang666/EasyUKey/issues) 中搜索类似问题
2. 如果没有找到，创建新的issue
3. 详细描述问题，包括：
   - 操作系统和版本
   - Go版本
   - 错误日志
   - 复现步骤

#### 功能请求
1. 在 [GitHub Issues](https://github.com/hang666/EasyUKey/issues) 中创建功能请求
2. 描述期望的功能和使用场景
3. 解释为什么这个功能对项目有价值

### 💻 代码贡献

#### 开发流程

1. **Fork项目**
```bash
# 在GitHub上Fork项目
# 克隆你的Fork
git clone https://github.com/your-username/EasyUKey.git
cd EasyUKey
```

2. **创建功能分支**
```bash
git checkout -b feature/amazing-feature
# 或者修复分支
git checkout -b fix/bug-description
```

3. **开发和测试**
```bash
# 编写代码
# 运行测试确保没有破坏现有功能
go test ./...
# 运行代码格式化
go fmt ./...
# 运行静态分析
go vet ./...
```

4. **提交更改**
```bash
git add .
git commit -m "feat: 添加令人惊叹的功能"
# 使用约定式提交格式
```

5. **推送和创建PR**
```bash
git push origin feature/amazing-feature
# 在GitHub上创建Pull Request
```

#### 提交信息规范

我们使用[约定式提交](https://www.conventionalcommits.org/)格式：

```
<类型>[可选作用域]: <描述>

[可选正文]

[可选脚注]
```

**类型说明：**
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式修改
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建或辅助工具更改

**示例：**
```bash
feat(auth): 添加生物识别认证支持
fix(client): 修复USB设备识别失败问题
docs: 更新API文档和使用示例
```

### 📋 代码规范

#### Go代码规范

1. **遵循Go官方代码风格**
```bash
# 使用gofmt格式化代码
go fmt ./...

# 使用goimports管理导入
goimports -w .
```

2. **错误处理**
```go
// 正确的错误处理方式
result, err := someFunction()
if err != nil {
    return fmt.Errorf("操作失败: %w", err)
}
```

3. **注释规范**
```go
// Package auth 提供身份认证相关功能
package auth

// AuthService 认证服务结构体
type AuthService struct {
    // db 数据库连接
    db *gorm.DB
}

// StartAuth 开始认证流程
// 参数 userID: 用户标识
// 参数 req: 认证请求
// 返回 认证响应和错误信息
func (s *AuthService) StartAuth(userID string, req *AuthRequest) (*AuthResponse, error) {
    // 实现逻辑
}
```

#### 测试规范

1. **单元测试**
```go
func TestAuthService_StartAuth(t *testing.T) {
    tests := []struct {
        name    string
        userID  string
        req     *AuthRequest
        want    *AuthResponse
        wantErr bool
    }{
        {
            name:   "正常认证",
            userID: "test_user",
            req:    &AuthRequest{Challenge: "test"},
            want:   &AuthResponse{Success: true},
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

2. **集成测试**
```go
func TestAuthIntegration(t *testing.T) {
    // 设置测试环境
    testDB := setupTestDB(t)
    defer teardownTestDB(t, testDB)
    
    // 执行集成测试
}
```

### 🔍 代码审查

#### PR审查清单

**功能性：**
- [ ] 功能是否按预期工作
- [ ] 是否有足够的测试覆盖
- [ ] 是否处理了边界情况
- [ ] 错误处理是否恰当

**代码质量：**
- [ ] 代码是否清晰易读
- [ ] 是否遵循项目约定
- [ ] 是否有适当的注释
- [ ] 是否有性能问题

**安全性：**
- [ ] 是否有安全漏洞
- [ ] 敏感信息是否正确处理
- [ ] 输入验证是否充分

**文档：**
- [ ] 是否更新了相关文档
- [ ] API变更是否记录
- [ ] 示例代码是否正确

### 🏆 贡献者认可

#### 贡献者类型

- **代码贡献者**：提交代码、修复bug、添加功能
- **文档贡献者**：改进文档、翻译、教程编写
- **测试贡献者**：编写测试、性能测试、安全测试
- **设计贡献者**：UI/UX设计、架构设计
- **社区贡献者**：问题回答、社区管理、推广

#### 致谢方式

- 代码贡献者将在项目README中列出
- 重要贡献者将获得项目徽章
- 优秀贡献可能被邀请成为项目维护者

### 📞 联系方式

- **GitHub Issues**: 功能请求和bug报告
- **GitHub Discussions**: 一般讨论和问答
- **Email**: hang666@example.com（维护者联系方式）

---

感谢您对EasyUKey项目的关注和贡献！每一个贡献都让项目变得更好。🙏

## 📄 许可证

本项目采用MIT许可证。详情请参阅 [LICENSE](LICENSE) 文件。

### 许可证要点

- ✅ **商业使用**：可用于商业项目
- ✅ **修改**：可以修改源代码
- ✅ **分发**：可以分发原始或修改后的代码
- ✅ **私人使用**：可用于私人项目
- ⚠️ **责任**：使用本软件的风险由用户承担
- ⚠️ **保证**：软件"按原样"提供，不提供任何保证

### 第三方许可证

本项目使用了以下开源组件：

| 组件 | 许可证 | 用途 |
|------|--------|------|
| Echo | MIT | Web框架 |
| GORM | MIT | ORM库 |
| Gorilla WebSocket | BSD-2-Clause | WebSocket支持 |
| Go-JWT | MIT | JWT处理 |
| Crypto | BSD-3-Clause | 加密算法 |

## 📞 支持与反馈

### 📚 文档资源

- **[API文档](https://github.com/hang666/EasyUKey/wiki/API-Reference)**：完整的API参考
- **[用户指南](https://github.com/hang666/EasyUKey/wiki/User-Guide)**：详细的使用教程
- **[开发者文档](https://github.com/hang666/EasyUKey/wiki/Developer-Guide)**：开发者集成指南
- **[部署指南](https://github.com/hang666/EasyUKey/wiki/Deployment)**：生产环境部署

### 🆘 获取帮助

#### GitHub Issues
- **[报告Bug](https://github.com/hang666/EasyUKey/issues/new?template=bug_report.md)**
- **[功能请求](https://github.com/hang666/EasyUKey/issues/new?template=feature_request.md)**
- **[一般问题](https://github.com/hang666/EasyUKey/issues/new?template=question.md)**

#### 社区支持
- **[GitHub Discussions](https://github.com/hang666/EasyUKey/discussions)**：社区讨论和问答
- **[Wiki](https://github.com/hang666/EasyUKey/wiki)**：详细文档和教程

#### 企业支持
如需企业级支持，请联系：
- **邮箱**: support@easyukey.com
- **技术支持**: tech@easyukey.com

### 📊 项目统计

![GitHub stars](https://img.shields.io/github/stars/hang666/EasyUKey?style=social)
![GitHub forks](https://img.shields.io/github/forks/hang666/EasyUKey?style=social)
![GitHub issues](https://img.shields.io/github/issues/hang666/EasyUKey)
![GitHub pull requests](https://img.shields.io/github/issues-pr/hang666/EasyUKey)

### 🌟 致谢

特别感谢以下贡献者和支持者：

- **核心开发团队**：hang666
- **社区贡献者**：感谢所有提交PR和报告问题的贡献者
- **测试用户**：感谢早期用户的反馈和建议

---

<div align="center">

**EasyUKey** - 让身份验证更简单、更安全！🔐✨

[![Star History Chart](https://api.star-history.com/svg?repos=hang666/EasyUKey&type=Date)](https://star-history.com/#hang666/EasyUKey&Date)

</div>
