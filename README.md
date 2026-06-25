# PRTS Robot — Discord 角色扮演机器人

基于《明日方舟》角色设定的 Discord AI 聊天机器人，接入 DeepSeek 大模型，支持 SKILL.md 角色扮演、SQLite 持久化会话、Docker 一键部署。

## 特性

- 🎭 **角色扮演**：SKILL.md 作为 system prompt，不同频道可绑定不同角色
- 🔧 **角色生成**：从 PRTS Wiki 页面一键生成角色设定（P2）
- 🛠️ **工具调用**：Function Calling，角色可按需调用外部 API（P3）
- 💾 **SQLite**：全量会话历史持久化，窗口读取
- 🐳 **Docker**：单容器部署，GitHub Actions 自动构建镜像

## 快速开始

### 1. Discord 创建 Bot

[Discord Developer Portal](https://discord.com/developers/applications) → New Application → Bot → Reset Token

### 2. 配置

```bash
cp config.example.yaml config.yaml
echo 'DISCORD_BOT_TOKEN=xxx' > .env
echo 'DEEPSEEK_API_KEY=sk-xxx' >> .env
```

### 3. 启动

```bash
docker compose pull && docker compose up -d
```

### 4. 邀请入服

Developer Portal → OAuth2 → URL Generator → 勾 `bot` + `Send Messages` + `Read Message History` → 链接邀请

## 使用

### 触发模式

| 模式 | 说明 |
|------|------|
| `mention` | 仅 @机器人时回复（默认） |
| `all` | 所有消息都回复 |

## 项目结构

```
├── cmd/bot/main.go          # 入口
├── internal/
│   ├── core/                 # Discord 连接 + 配置
│   ├── message/              # 消息类型
│   ├── session/              # SQLite 会话管理
│   ├── llm/                  # DeepSeek 客户端
│   └── persona/              # 角色系统 (P2)
├── prompts/                  # 角色生成规则
├── tools/                    # Python 角色生成工具
├── data/personas/            # 角色 SKILL 文件
├── config.example.yaml
├── docker-compose.yml
└── docs/
```

## 技术栈

| 层次 | 选型 |
|------|------|
| 语言 | Go 1.25 |
| 协议 | Discord Gateway (discordgo) |
| 大模型 | DeepSeek V4 Flash |
| 数据库 | SQLite |
| 部署 | Docker + GitHub Actions |

## 阶段

| 阶段 | 内容 | 状态 |
|------|------|------|
| P1 | Discord 对话 + SQLite 会话 | ✅ 完成 |
| P2 | 角色系统 + 命令 + 生成器 | 🔜 |
| P3 | Agent 工具调用 | 📋 |
| P4 | RAG 知识库 | 📋 |

## 文档

- [设计规格](docs/bot-design.md)
- [部署指南](docs/deploy.md)

## 许可证

MIT
