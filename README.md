# EasyUKey

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Development Status](https://img.shields.io/badge/status-Development-orange)](https://github.com/hang666/EasyUKey)

EasyUKey 是一个基于USB设备的高安全性认证服务组件，提供简易U盾实现。采用客户端-服务器架构，使用实时WebSocket通信，集成多重加密保护和TOTP双因子认证。

> ⚠️ **提醒**：本项目目前仍在开发阶段，基本功能已经实现，但功能和安全性依旧持续改进中，请谨慎用于生产环境。

## 🚀 核心特性

### 🔐 多重安全防护

- **硬件身份验证**：基于USB设备的物理身份验证
- **双重设备识别**：同时验证U盘分区序列号和设备序列号，确保硬件唯一性
- **OnceKey防复制**：动态一次性密钥机制，有效防止硬件复制攻击
- **多层加密保护**：
  - ECDH密钥交换（P-256曲线）
  - AES-256-GCM端到端加密
  - PIN码 + 加密密钥的安全存储
- **TOTP双因子认证**：支持时间基础的一次性密码验证
- **签名验证**：回调请求HMAC-SHA256签名防篡改

### 🌐 实时通信架构

- **WebSocket Hub**：高并发连接管理，支持单点登录策略
- **消息加密**：所有通信消息支持端到端加密
- **心跳检测**：智能连接状态监控和自动重连
- **设备状态同步**：实时设备在线状态同步

### 🔧 开发友好

- **Go SDK**：完整的Go语言SDK，支持认证、管理等所有功能
- **RESTful API**：标准化的HTTP API接口
- **异步回调**：支持认证结果异步回调通知

## 📋 验证方案

### 认证流程

1. **应用系统发起认证**：第三方应用向EasyUKey服务器提交认证请求
2. **服务器转发请求**：EasyUKey服务器将认证请求转发给对应的USB客户端
3. **双重硬件识别**：系统同时验证U盘分区序列号和设备序列号，确保硬件唯一性
4. **生成认证密钥**：客户端基于硬件信息和OnceKey生成认证密钥
5. **返回认证结果**：认证结果通过加密通道返回给EasyUKey服务器
6. **OnceKey交换确认**：客户端与服务器进行新的一次性密钥交换，更新防复制密钥
7. **确认成功**：客户端向服务器发送确认请求，服务器确认成功
8. **异步回调通知**：服务器向应用系统发送认证结果的回调通知

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

## 🎯 使用场景

### 企业级应用

- **OA系统登录**：替代传统密码登录方式
- **财务系统**：高安全性的金融交易认证
- **数据中心**：服务器和设备的物理访问控制

### 开发集成

- **Web应用**：集成到现有的Web应用认证流程
- **桌面应用**：本地应用程序的安全认证

## 📝 TODO

- [ ] 实现多平台支持
- [x] client端使用pin+encryption_key加密存储信息
- [x] 通信加密
- [x] 异步回调
- [ ] 完善认证流程细节
- [ ] 客户端认证接口同步返回结果
- [ ] 完善管理页面
- [ ] 完善错误处理和日志
- [ ] 完善文档

## 🤝 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'feat: Add some AmazingFeature'`)
4. 推送分支 (`git push origin feature/AmazingFeature`)
5. 创建Pull Request

## 📄 许可证

本项目采用MIT许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 📞 支持

- **GitHub Issues**：[提交问题](https://github.com/hang666/EasyUKey/issues)
- **文档**：[详细文档](https://github.com/hang666/EasyUKey/wiki)

---

**EasyUKey** - 让身份验证更简单、更安全！🔐✨
