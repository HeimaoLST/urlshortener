# JumpURL - 高性能、智能化的开源短链接管理平台
 JumpURL是一个基于 Go 语言构建的现代化、高性能的开源短链接服务。它不仅仅是一个 URL 缩短工具，更是一个为个人、开发者和中小型企业设计的，集品牌化、智能化、数据私有化于一体的链接管理平台。


### ✨ 核心特性 (Key Features)

- **高性能与高并发**:
  - 基于 Go + Gin 框架，提供毫秒级的重定向响应。
  - 创新的**异步点击数统计**架构，利用 Channel 和后台 Goroutine 批量处理数据库更新，轻松应对高并发流量冲击。
- **智能化链接管理 (AI Powered)**:
  - **AI 智能命名**: 创建链接时，可调用大语言模型（LLM）分析目标网页内容，智能推荐 SEO 友好的自定义短码。
  - **(规划中) AI 分析报告**: 将原始点击数据转化为易于理解的趋势分析和洞察报告。
- **企业级缓存策略**:
  - 采用 **Redis Hash (HSET)** 对所有链接数据进行缓存，相比传统 `SET` 命令，**极大降低了内存占用**。
  - 通过在应用层**模拟字段级 TTL**，实现了对每个短链接缓存周期的精细化控制。
- **品牌化与定制**:
  - 支持**自定义短码**，让你的链接更具辨识度。
  - 支持绑定**私有域名**，提升品牌形象和用户信任度。
- **数据私有化**:
  - 可完全部署在您自己的服务器上，所有数据（链接、点击记录）100% 由您掌控。
- **现代化技术栈**:
  - 使用 **sqlc** 自动生成类型安全的数据库操作代码，兼顾开发效率与 SQL 性能。
  - 提供 **Docker & Docker Compose** 支持，实现一键启动和便捷部署。

### 🚀 架构概览 (Architecture Overview)

JumpURL 采用清晰、可扩展的后端架构设计，核心组件包括：
 


### 🔧 技术栈 (Technology Stack)

- **后端**: Go 1.22+
- **Web 框架**: Gin
- **数据库**: PostgreSQL
- **缓存**: Redis
- **数据库工具**: sqlc
- **部署**: Docker, Docker Compose

### 🏃‍ 如何开始 (Getting Started)

在几分钟内即可启动并运行您自己的 JumpURL 服务。

#### 1. 前提条件

- 已安装 [Docker](https://www.docker.com/get-started) 和 [Docker Compose](https://docs.docker.com/compose/install/)。

#### 2. 配置

1. 克隆本仓库到您的本地机器：

   ```
   git clone https://github.com/heimaolst/JumpURL.git
   cd JumpURL
   ```

2. 创建一个 `.env` 配置文件，可以从 `env.example` 复制：

   ```
   cp env.example .env
   ```

3. 根据您的需求修改 `.env` 文件。至少需要配置数据库密码等信息。

#### 3. 启动服务

在项目根目录下，执行以下命令：

```
docker-compose up -d
```

该命令会以后台模式启动 JumpURL 应用、PostgreSQL 数据库和 Redis 缓存。服务将默认在 `http://localhost:8080` 上可用。

### 📖 API 参考 (API Reference)

- `POST /api/create`: 创建一个新的短链接。
- `GET /api/jump:shortcode`: 执行重定向。
- ... (更多接口请参考 API 文档)

### 🤝 如何贡献 (Contributing)

我们非常欢迎来自社区的贡献！无论是提交 Bug、建议新功能还是直接贡献代码。

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交一个 Pull Request

### 📄 授权许可 (License)

本项目基于 [MIT License](https://opensource.org/licenses/MIT) 进行授权。
