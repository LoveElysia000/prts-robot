# PRTS Robot — QQ 群聊 AI 机器人

基于 Go 的 QQ 群聊 AI 机器人，接入 DeepSeek 大模型（OpenAI 兼容格式），支持角色扮演对话、SQLite 持久化会话，Docker 一键部署。使用 **NapCat + OneBot v11 协议** 与 QQ 通信。

## 特性

- 🤖 **AI 对话**：基于 DeepSeek V4 Flash，支持上下文记忆（全量存储、窗口读取）
- 🎭 **角色扮演**：按群绑定角色，SKILL.md 作为 system prompt（P2）
- 🛠️ **Agent 工具调用**：Function Calling 自动调用外部工具（P3）
- 📚 **RAG 知识库**：文档上传 + 向量检索增强问答（P4）
- 💾 **SQLite 持久化**：会话历史全量保存，重启不丢
- 🐳 **Docker 部署**：一条命令启动，无需手动配置环境

## 快速开始

### 前置条件

1. 安装 Docker 和 Docker Compose
2. 在 [DeepSeek 开放平台](https://platform.deepseek.com) 获取 API Key

### 配置

编辑 `config.yaml`：

```yaml
napcat:
  access_token: "your-access-token"

deepseek:
  api_key: "sk-你的APIKey"
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  default_system_prompt: "你是一个友好的QQ群助手"

trigger:
  mode: "hybrid"          # all | at | hybrid
  command_prefix: "/"

database:
  path: "./data/bot.db"
```

### 启动

```bash
docker compose up -d
```

### 首次运行 NapCat

启动后 NapCat 容器会输出二维码，扫描登录 QQ 账号：

```bash
docker compose logs -f napcat
```

登录完成后，在群里 @机器人 即可开始对话。

## 触发模式

| 模式 | 说明 |
|------|------|
| `all` | 群内所有消息都回复 |
| `at` | 仅 @机器人 的消息才回复 |
| `hybrid` | @机器人才回复（当前默认） |

## 项目结构

```
robot/
├── cmd/bot/main.go          # 主入口
├── internal/
│   ├── core/
│   │   ├── config.go        # 配置加载
│   │   ├── bot.go           # ZeroBot 服务器 + 消息处理管线
│   │   └── qqapi.go         # OneBot v11 API 封装
│   ├── message/
│   │   ├── types.go         # 消息结构体
│   │   └── handler.go       # 触发判断
│   ├── session/
│   │   └── manager.go       # SQLite 会话管理
│   └── llm/
│       └── client.go        # DeepSeek 客户端
├── config.yaml              # 配置文件
├── Dockerfile
├── docker-compose.yml
├── napcat/                  # NapCat 配置目录
└── docs/                    # 设计文档 + 实现计划
```

## 技术栈

| 层次 | 选型 |
|------|------|
| 语言 | Go 1.22+ |
| QQ 协议 | NapCat + OneBot v11（ZeroBot SDK） |
| 大模型 | DeepSeek V4 Flash（go-openai SDK） |
| 数据库 | SQLite（modernc.org/sqlite，纯 Go） |
| 配置 | YAML（gopkg.in/yaml.v3） |
| 日志 | log/slog（标准库） |
| 部署 | Docker + Docker Compose |

## 开发

```bash
# 克隆项目
git clone <你的仓库地址>
cd robot

# 运行测试
go test ./...

# 编译
go build -o bot ./cmd/bot

# 静检
go vet ./...
```

## 阶段规划

| 阶段 | 内容 | 状态 |
|------|------|------|
| P1 | 核心对话（ZeroBot/NapCat + DeepSeek + SQLite） | ✅ 已完成 |
| P2 | 角色系统 + 命令插件 + 角色生成器 | 🔜 计划中 |
| P3 | Agent 工具调用（Function Calling） | 📋 设计中 |
| P4 | RAG 知识库（Qdrant + 智谱 Embedding） | 📋 设计中 |

## 设计文档

完整设计文档见：
- [设计规格](docs/bot-design.md)
- [P1 实现计划（NapCat）](docs/p1-plan-napcat.md)

## 许可证

MIT
