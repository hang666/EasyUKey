# EasyUKey

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Development Status](https://img.shields.io/badge/status-Development-orange)](https://github.com/hang666/EasyUKey)

**将任何U盘变为您的专属安全密钥 (U盾)，无需特定硬件，即插即用。**

EasyUKey 是一个基于USB设备的开源高安全性认证服务解决方案，提供简易U盾实现。采用客户端-服务器架构，使用实时WebSocket通信，集成多重加密保护和TOTP双因子认证。

> ⚠️ **提醒**：本项目目前仍在开发阶段，基本功能已经实现，但功能和安全性依旧持续改进中，请谨慎用于生产环境。

## ✨ 核心亮点

* **极致便利**：将您随身携带的任何U盘转变为一个硬件U盾。无需购买或等待特定硬件。
* **高安全性**：为您的敏感操作提供一层坚固的物理安全保障，有效防止未经授权的访问。
* **开源透明**：所有代码完全开源，由社区驱动，安全、可信赖。
* **易于集成**：可以轻松地集成到您现有的服务或应用中，作为多因素认证（MFA）的一环。

## 🚀 它是如何工作的？

EasyUKey 通过在您的U盘上创建一个唯一的、经过加密的密钥文件来识别您的身份。

1. **初始化**: 在您的U盘上生成一个安全的密钥。
2. **认证**: 当需要进行安全认证时，系统会提示您插入U盘。
3. **验证**: EasyUKey 会验证U盘上的密钥是否有效，验证通过后即可完成操作。

整个过程就像使用银行U盾一样简单，但硬件载体却是您最常见的U盘。

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

* **数据库**：MySQL 5.7+
* **依赖**：Go, Git, Make
* **Docker部署**：Docker, Docker Compose

### 安装部署

#### 方式一：传统部署

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

#### 方式二：Docker部署

1. **克隆项目**

```bash
git clone https://github.com/hang666/EasyUKey.git
cd EasyUKey
```

2. **配置环境变量**

```bash
# 复制环境变量示例文件
cp .env.example .env

# 编辑.env文件，设置必需的EASYUKEY_SECURITY_ENCRYPTION_KEY
# 可以使用以下命令生成32位随机密钥：
# openssl rand -hex 32
```

3. **选择部署方式**

**使用外部MySQL（推荐用于生产环境）：**

```bash
# 编辑.env文件，配置外部数据库连接信息：
# EASYUKEY_DATABASE_HOST=your-mysql-host
# EASYUKEY_DATABASE_PASSWORD=your-mysql-password
# 等等...

# 启动服务（使用外部MySQL数据库）
docker-compose up -d
```

**使用内置MySQL（推荐用于开发测试）：**

```bash
# 启动服务（包含MySQL数据库）
docker-compose -f docker-compose.db.yml up -d
```

4. **验证部署**

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f server
```

服务启动后可访问：<http://localhost:8888/admin> 管理页面

### 客户端部署

1. 构建客户端

```bash
# 构建客户端
make client ENCRYPT_KEY_STR=123456789 SERVER_ADDR=http://localhost:8888
```

2. 将构建好的客户端复制到USB设备打开即可使用

## 🎯 使用场景

### 企业级应用

* **OA系统登录**：替代传统密码登录方式
* **财务系统**：高安全性的金融交易认证
* **数据中心**：服务器和设备的物理访问控制

### 开发集成

* **Web应用**：集成到现有的Web应用认证流程
* **桌面应用**：本地应用程序的安全认证

## 📝 TODO

* [ ] 实现多平台支持
* [x] client端使用pin+encryption_key加密存储信息
* [x] 通信加密
* [x] 异步回调
* [ ] 完善认证流程细节
* [x] 客户端认证接口同步返回结果
* [ ] 完善管理页面
* [x] 完善错误处理
* [x] 完善日志
* [ ] 完善文档

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

* **GitHub Issues**：[提交问题](https://github.com/hang666/EasyUKey/issues)
* **文档**：[详细文档](https://github.com/hang666/EasyUKey/wiki)

---

**EasyUKey** - 让身份验证更简单、更安全！🔐✨
