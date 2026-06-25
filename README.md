# Lin (林) — Discord 角色扮演机器人

Discord AI 聊天机器人，接入 DeepSeek 大模型，支持角色扮演、SQLite 持久化、Docker 部署。

## 特性

- 🤖 **角色扮演**：SKILL.md 作为 system prompt，支持按频道绑定不同角色
- 🎭 **角色生成**：从 PRTS Wiki 一键生成角色设定（P2）
- 🛠️ **工具调用**：Function Calling（P3）
- 💾 **SQLite**：会话历史持久化，重启不丢
- 🐳 **Docker**：一条命令部署

## 快速开始

### 1. Discord 创建 Bot

`https://discord.com/developers/applications` → New Application → Bot → Reset Token

### 2. 配置

```bash
cp config.example.yaml config.yaml
# edit trigger mode (mention / all)

echo 'DISCORD_BOT_TOKEN=xxx' > .env
echo 'DEEPSEEK_API_KEY=sk-xxx' >> .env
```

### 3. 启动

```bash
docker compose pull && docker compose up -d
```

### 4. 邀请 Bot

Discord Developer Portal → OAuth2 → URL Generator → 勾 `bot` → 链接邀请入服务器

## 触发模式

| 模式 | 说明 |
|------|------|
| `mention` | 仅 @机器人 时回复（默认） |
| `all` | 所有消息都回复 |

## 项目结构

```
├── cmd/bot/main.go          # 入口
├── internal/
│   ├── core/
│   │   ├── config.go        # 配置加载
│   │   └── bot.go           # Discord 连接 + 消息处理
│   ├── message/
│   │   └── types.go         # 消息结构体
│   ├── session/
│   │   └── manager.go       # SQLite 会话管理
│   ├── llm/
│   │   └── client.go        # DeepSeek 客户端
│   └── persona/             # 角色系统 (P2)
├── data/personas/           # 角色文件
├── config.example.yaml
├── docker-compose.yml
├── Dockerfile
└── docs/                    # 设计文档
```

## 技术栈

| 层次 | 选型 |
|------|------|
| 语言 | Go 1.25 |
| 协议 | Discord 官方 API (discordgo) |
| 大模型 | DeepSeek V4 Flash |
| 数据库 | SQLite |
| 部署 | Docker |

## 文档

- [设计规格](docs/bot-design.md)
- [部署指南](docs/deploy.md)

## 许可证

MIT
