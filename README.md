# PRTS Robot — Discord 角色扮演机器人

基于《明日方舟》角色设定的 Discord AI 聊天机器人，接入 DeepSeek 大模型，支持 SKILL.md 角色扮演、SQLite 持久化会话、Docker + watchtower 自动部署。

## 特性

- 🎭 **角色扮演**：SKILL.md 作为 system prompt，不同频道可绑定不同角色
- 🔧 **角色生成**：从 PRTS Wiki 页面一键生成角色设定
- ✏️ **角色校正**：通过 AI 指令修正角色 persona
- 🛠️ **工具调用**：Function Calling，角色可按需调用外部 API（规划中）
- 💾 **SQLite**：全量会话历史持久化，对话成对写入，保证一致性
- ⏱️ **超时容错**：75s 外层 deadline，30s 单次 API 调用超时 + 退避重试，shutdown 级联取消
- 🐳 **Docker**：单容器部署，GitHub Actions 自动构建 + watchtower 自动更新

## 快速开始

### 1. Discord 创建 Bot

[Discord Developer Portal](https://discord.com/developers/applications) → New Application → Bot → Reset Token

### 2. 配置

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml 填入 api_key
echo 'DISCORD_BOT_TOKEN=xxx' > .env
echo 'DEEPSEEK_API_KEY=sk-xxx' >> .env
echo 'GHCR_TOKEN=ghp_xxx' >> .env   # GitHub PAT，给 watchtower 拉镜像用
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

### 命令

| 命令 | 说明 |
|------|------|
| `/角色 列表` | 查看所有角色 |
| `/角色 切换 <名字>` | 切换频道绑定的角色 |
| `/角色 重载` | 热加载 personas.yaml |
| `/角色校正 <名字> <指令>` | AI 修正角色 persona |
| `/生成角色 <slug> <Wiki URL>` | 从 PRTS Wiki 生成角色 |
| `/help` | 查看所有命令 |

## 项目结构

```
├── cmd/bot/main.go              # 入口
├── internal/
│   ├── core/                     # Discord 连接、消息处理、WorkerPool
│   │   ├── bot.go                # 核心 Bot 逻辑
│   │   ├── config.go             # 配置加载
│   │   └── worker.go             # 并发调度 WorkerPool
│   ├── llm/                      # DeepSeek API 客户端
│   │   └── client.go
│   ├── session/                  # SQLite 会话持久化
│   │   └── manager.go
│   └── persona/                  # 角色系统
│       ├── manager.go            # 角色加载 & 频道绑定
│       ├── card.go               # Persona 数据结构
│       ├── corrector.go          # AI 校正角色
│       └── generator/            # 角色生成器
│           ├── generator.go      # 4 层 LLM 生成
│           ├── fetcher.go        # Wiki HTML 抓取
│           ├── parser_bridge.go  # Python 解析桥接
│           └── writer.go         # SKILL.md 拼装
├── prompts/                      # LLM prompt 模板
├── tools/                        # Python 辅助脚本
├── data/
│   ├── bot.db                    # SQLite 数据库（运行时）
│   ├── personas.yaml             # 角色注册 & 频道绑定
│   └── personas/                 # 各角色的 SKILL.md 文件
├── config.example.yaml
├── docker-compose.yml            # bot + watchtower
├── Dockerfile                    # 多阶段构建 (Go + Python)
└── docs/
```

## 超时设计

```
shutdownCtx ──WithTimeout(75s)──▶ submitCtx ──▶ task.ctx ──▶ callLLM(ctx)
                                                              │
                                            Chat(llmCtx, 30s) │ 嵌套 WithTimeout
                                            超时? → ctx 活着? → 退避 1s → 重试 30s
```

- 外层 75s 是总预算，内层 30s 是单次调用上限，context 自动取更早的 deadline
- 重试跟随父 context，shutdown / Submit 超时自动取消
- 退避 1s 避免对已故障的链路秒重试
- WorkerPool 传播 Submit context 到 handler，Submit 返回时 cancel 孤儿任务

## 技术栈

| 层次 | 选型 |
|------|------|
| 语言 | Go |
| 协议 | Discord Gateway (discordgo) |
| 大模型 | DeepSeek (OpenAI 兼容 API) |
| 数据库 | SQLite (modernc.org/sqlite) |
| 部署 | Docker + GitHub Actions + Watchtower |

## 阶段

| 阶段 | 内容 | 状态 |
|------|------|------|
| P1 | Discord 对话 + SQLite 会话 | ✅ 完成 |
| P2 | 角色系统 + 命令 + 生成器 + 校正 | ✅ 完成 |
| P3 | Agent 工具调用 | 📋 规划中 |
| P4 | RAG 知识库 | 📋 规划中 |

## 文档

- [设计规格](docs/bot-design.md)
- [部署指南](docs/deploy.md)

## 许可证

MIT
