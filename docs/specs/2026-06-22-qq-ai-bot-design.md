# QQ AI 机器人设计文档

## 概述

手写一个基于 Go 的 QQ 聊天机器人，接入 DeepSeek 大模型，支持 AI 对话与功能型插件。采用分阶段递进开发，共四个阶段（P1-P4），每阶段独立可运行。

## 需求摘要

| 维度 | 决策 |
|------|------|
| 平台 | QQ（NapCat 协议端，OneBot v11） |
| 语言 | Go |
| 触发方式 | 私聊始终回复；群聊可选「@才回复」或「全量回复」，可配置 |
| 上下文记忆 | 全量历史，按会话隔离，持久化存储 |
| 大模型 | DeepSeek（deepseek-chat，OpenAI 兼容格式） |
| 功能范围 | AI 对话 + 娱乐小工具 + 群管理 + Agent 工具调用 + RAG 知识库 |
| 部署 | Docker + Docker Compose |

## 技术栈

| 层次 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.22+ | goroutine 并发处理消息 |
| 依赖管理 | Go modules | 标准 go.mod |
| 协议端 | NapCat (Docker) | QQ 协议实现，OneBot v11 接口 |
| OneBot 协议库 | `github.com/wdvxdr1123/zerobot` | OneBot v11 Go 框架，活跃维护 |
| 大模型 SDK | `github.com/sashabaranov/go-openai` | OpenAI 兼容，可直接用于 DeepSeek |
| 配置 | `gopkg.in/yaml.v3` | YAML 配置文件 |
| 日志 | `log/slog`（标准库） | 结构化日志 |
| 向量库 (P4) | Qdrant + `github.com/qdrant/qdrant-go` | 官方 Go SDK，Docker 部署 |
| 文档解析 (P4) | `github.com/ledongthuc/pdf` + 标准库 | PDF/MD/TXT 解析 |
| 容器化 | Docker + Docker Compose | NapCat + Bot (+ Qdrant) 容器编排 |

### 选型理由

- **Go 而非 Python**：开发者有 Go 背景；部署产物为单二进制，镜像体积小；goroutine 天然契合多会话并发。
- **ZeroBot 而非手写 WebSocket**：OneBot 协议层无需重复造轮子，ZeroBot 活跃维护，业务逻辑完全自控。
- **go-openai 而非手写 HTTP**：DeepSeek 兼容 OpenAI 格式，官方 SDK 封装完善，省去手写请求/响应解析。
- **Qdrant 而非 ChromaDB**：ChromaDB 无 Go SDK，Qdrant 有官方 Go SDK 且支持 Docker 部署，与 Go 配合最顺。
- **P4 文档解析**：先支持 Markdown/TXT（标准库即可），PDF 用 `ledongthuc/pdf`，按需启用。

## 整体架构

```
┌─────────────────────────────────────────────────────┐
│  QQ 客户端                                           │
└──────────┬──────────────────────────────────────────┘
           │ OneBot v11 协议 (WebSocket)
┌──────────▼──────────────────────────────────────────┐
│  NapCat (协议端, 独立容器)                            │
└──────────┬──────────────────────────────────────────┘
           │ WebSocket 反向连接
┌──────────▼──────────────────────────────────────────┐
│  Bot 核心 (Go)                                       │
│                                                      │
│  ┌─────────────┐   ┌──────────────┐                 │
│  │ 消息接收/解析 │──▶│ 消息处理管线  │                 │
│  └─────────────┘   └──────┬───────┘                 │
│                           │                          │
│         ┌─────────────────┼─────────────────┐        │
│         ▼                 ▼                 ▼        │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐   │
│  │ 命令路由    │   │ AI 对话     │   │ RAG 检索   │   │
│  │ (P2)       │   │ (P1/P3)    │   │ (P4)       │   │
│  └─────┬──────┘   └─────┬──────┘   └─────┬──────┘   │
│        │                │                │           │
│        ▼                ▼                ▼           │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐   │
│  │ 插件集合    │   │ DeepSeek   │   │ Qdrant     │   │
│  │ 娱乐/群管   │   │ +Agent工具 │   │ 向量库     │   │
│  └────────────┘   └────────────┘   └────────────┘   │
└─────────────────────────────────────────────────────┘
```

### 消息流转

1. NapCat 收到 QQ 消息，通过 WebSocket 推送给 Bot
2. Bot 解析消息（文本/@/图片等）
3. 消息管线判断触发条件：是否需要回复
4. 判断消息类型：命令（带前缀 `/`）→ 命令路由；否则 → AI 对话管线
5. AI 对话管线：组装上下文 → 调用 DeepSeek →（P3）可能触发工具调用 → 生成回复
6. 通过 NapCat 发送回复

## 项目目录结构（全阶段）

```
robot/
├── cmd/
│   └── bot/
│       └── main.go              # 入口
├── internal/
│   ├── core/
│   │   ├── bot.go               # 主逻辑：ZeroBot 初始化与事件注册
│   │   └── config.go            # 配置加载与校验
│   ├── message/
│   │   ├── handler.go           # 消息处理管线（触发判断 + 分流）
│   │   ├── parser.go            # OneBot 消息段解析
│   │   └── sender.go            # 回复封装与发送
│   ├── session/
│   │   └── manager.go           # 会话与上下文管理（全量历史）
│   ├── llm/
│   │   ├── client.go            # DeepSeek API 客户端 (P1)
│   │   ├── agent.go             # Function Calling 调度 (P3)
│   │   └── tools/               # Agent 工具目录 (P3)
│   │       ├── registry.go      # 工具注册
│   │       ├── web_search.go
│   │       └── weather.go
│   ├── plugins/                 # 功能型插件 (P2)
│   │   ├── registry.go          # 插件注册与路由
│   │   ├── plugin.go            # 插件接口定义
│   │   ├── entertainment/       # 娱乐小工具
│   │   │   ├── fortune.go       # 今日运势
│   │   │   └── random_image.go  # 随机图片
│   │   └── groupadmin/          # 群管理
│   │       ├── welcome.go       # 入群欢迎
│   │       └── mute.go          # 禁言/踢人
│   └── rag/                     # RAG 知识库 (P4)
│       ├── indexer.go           # 文档解析与向量化
│       ├── retriever.go         # 检索 + 注入 prompt
│       └── store.go             # Qdrant 连接管理
├── data/                        # 运行时数据
│   ├── sessions.json            # 会话历史持久化
│   └── qdrant/                  # (P4) 向量库数据卷映射
├── config.yaml                  # 配置文件
├── go.mod
├── Dockerfile
├── docker-compose.yml           # NapCat + Bot (+ Qdrant) 容器编排
└── README.md
```

## 配置文件设计

```yaml
# config.yaml
napcat:
  ws_url: "ws://napcat:3001"        # NapCat WebSocket 地址
  access_token: "your-token"         # 鉴权 token

deepseek:
  api_key: "sk-xxx"
  base_url: "https://api.deepseek.com"
  model: "deepseek-chat"
  system_prompt: "你是一个友好的QQ群助手"

trigger:
  mode: "hybrid"          # all | at | hybrid
  command_prefix: "/"     # 命令前缀（P2 用）

session:
  persist_path: "./data/sessions.json"

# P4 扩展
qdrant:
  url: "http://qdrant:6333"
  collection: "knowledge"
  embedding_model: "text-embedding-3-small"  # 或 DeepSeek embedding
```

## 阶段设计

### P1：核心对话

**目标**：机器人上线，能进行带上下文的 AI 对话。

#### 组件职责

| 文件 | 职责 |
|------|------|
| `cmd/bot/main.go` | 入口，加载配置，启动 ZeroBot |
| `internal/core/bot.go` | ZeroBot 初始化，注册消息事件处理 |
| `internal/core/config.go` | 读取 config.yaml，结构体映射 |
| `internal/message/parser.go` | 解析 OneBot 消息段（文本/@/图片） |
| `internal/message/handler.go` | 判断触发条件，分流到 AI 管线 |
| `internal/message/sender.go` | 封装回复消息并发送 |
| `internal/session/manager.go` | 按 group_id/user_id 维护全量历史，持久化 |
| `internal/llm/client.go` | 封装 DeepSeek API 调用 |

#### 触发逻辑

```go
func ShouldReply(msg *Message, cfg *Config) bool {
    if msg.IsPrivate {
        return true                      // 私聊始终回复
    }
    switch cfg.Trigger.Mode {
    case "all":
        return true                      // 群消息全量回复
    case "at":
        return msg.IsAtBot               // 仅 @ 时回复
    case "hybrid":
        return msg.IsAtBot               // 私聊全回 + 群@才回（私聊已在上面处理）
    }
    return false
}
```

#### 会话管理（全量历史）

```go
// 会话 key 规则：群聊 "group_{group_id}"，私聊 "private_{user_id}"
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string][]Message    // key -> 全量历史
    persistPath string
}

// 读取历史
func (sm *SessionManager) Get(sessionKey string) []Message

// 追加一条并持久化
func (sm *SessionManager) Append(sessionKey string, msg Message) error
```

- 按 `group_{id}` / `private_{id}` 做 key 隔离
- 持久化到 `data/sessions.json`，重启不丢失
- 读写加锁，防止并发冲突
- P1 不做 token 截断（预留接口，后续可加）

#### DeepSeek 客户端

```go
type DeepSeekClient struct {
    client *openai.Client
    model  string
}

func (c *DeepSeekClient) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error)
```

#### P1 完成标准

- [ ] 机器人上线，NapCat 连接稳定（ZeroBot 内置断线重连）
- [ ] 私聊能正常对话，带上下文
- [ ] 群聊 @机器人 能对话，带上下文
- [ ] 会话历史持久化，重启不丢
- [ ] Docker Compose 一键启动（NapCat + Bot 双容器）

#### P1 依赖

```
github.com/wdvxdr1123/zerobot      # OneBot 协议
github.com/sashabaranov/go-openai  # DeepSeek (OpenAI 兼容)
gopkg.in/yaml.v3                    # 配置
# log/slog 为标准库，无需额外依赖
```

---

### P2：功能型插件（娱乐 + 群管）

**目标**：命令路由框架 + 插件系统，支持娱乐与群管功能。

#### 命令路由

```
用户输入 "/今日运势" → 命令路由 → 匹配 fortune 插件 → 执行 → 回复
用户输入 "你好"      → 非命令   → 走 AI 对话管线
```

- 消息以 `command_prefix`（默认 `/`）开头 → 命令路由
- 否则 → AI 对话管线

#### 插件接口

```go
// internal/plugins/plugin.go
type Plugin interface {
    Name() string
    Command() string        // 触发命令，如 "今日运势"
    Description() string    // 用于 /help
    Handle(ctx context.Context, args string, msg *Message) (string, error)
}
```

#### 插件注册

- 启动时在各插件包的 `init()` 中调用 `registry.Register()` 注册
- `/help` 命令自动列出所有已注册插件

#### 示例插件

| 插件 | 命令 | 功能 |
|------|------|------|
| 今日运势 | `/今日运势` | 随机生成运势分值 + 评语 |
| 随机图片 | `/随机图片` | 调用随机图 API 返回图片 |
| 入群欢迎 | （事件触发） | 新成员入群自动发欢迎语 |
| 禁言 | `/禁言 @用户 时长` | 调用 OneBot `set_group_ban` |
| 踢人 | `/踢人 @用户` | 调用 OneBot `set_group_kick` |

#### P2 完成标准

- [ ] 命令路由框架可用，`/help` 列出所有插件
- [ ] 2 个娱乐插件 + 2 个群管插件可用
- [ ] 新增插件只需实现接口并在 `init()` 注册

---

### P3：AI Agent 工具调用

**目标**：DeepSeek Function Calling 集成，AI 能调用外部工具。

#### 架构

```
用户: "今天北京天气怎么样？"
  ↓
AI 管线 → DeepSeek (Function Calling)
  ↓ 返回 tool_call: get_weather(city="北京")
Agent 调度器 → 执行 weather 工具 → 获取结果
  ↓ 将结果回传给 DeepSeek
DeepSeek → 生成最终回复："北京今天 28°C 晴..."
```

#### 工具定义

```go
// internal/llm/tools/weather.go
var WeatherTool = openai.Tool{
    Type: openai.ToolTypeFunction,
    Function: &openai.FunctionDefinition{
        Name:        "get_weather",
        Description: "查询指定城市天气",
        Parameters: map[string]any{
            "type": "object",
            "properties": map[string]any{
                "city": map[string]any{
                    "type":        "string",
                    "description": "城市名",
                },
            },
            "required": []string{"city"},
        },
    },
}

func ExecuteWeather(ctx context.Context, args map[string]any) (string, error)
```

#### Agent 调度循环

```go
// internal/llm/agent.go
func ChatWithTools(ctx context.Context, messages []openai.ChatCompletionMessage, tools []openai.Tool) (string, error) {
    for {
        resp, err := client.Chat(ctx, messages, tools)
        if err != nil {
            return "", err
        }
        if len(resp.ToolCalls) == 0 {
            return resp.Content, nil    // 无工具调用，返回最终回复
        }
        for _, call := range resp.ToolCalls {
            result, err := registry.Execute(ctx, call)
            if err != nil {
                result = "工具执行失败: " + err.Error()
            }
            messages = append(messages, toolResultMessage(call, result))
        }
        // 继续循环，模型可能继续调工具
    }
}
```

#### P3 完成标准

- [ ] DeepSeek Function Calling 集成
- [ ] 工具自动注册（类似 P2 插件机制）
- [ ] 2 个示例工具：网页搜索 + 天气查询
- [ ] 多轮工具调用支持（调一次工具后能继续调）

---

### P4：RAG 知识库

**目标**：文档上传、向量化、检索增强问答。

#### 架构

```
上传文档 → 解析分块 → 向量化(Embedding) → 存入 Qdrant

用户提问 → 问题向量化 → Qdrant 检索 Top-K → 注入 prompt → DeepSeek 生成
```

#### 文档处理

- 支持格式：Markdown / TXT（标准库）+ PDF（`ledongthuc/pdf`）
- 分块策略：按段落 + 字数上限（500 字），重叠 50 字
- Embedding：DeepSeek embedding API 或 OpenAI `text-embedding-3-small`

#### 检索流程

```go
// internal/rag/retriever.go
func RAGChat(ctx context.Context, question string, sessionKey string) (string, error) {
    // 1. 检索相关文档块
    docs, err := qdrant.Search(ctx, question, topK=3)
    if err != nil {
        return "", err
    }

    // 2. 注入 system prompt
    context := strings.Join(docs, "\n---\n")
    messages := buildRAGMessages(context, sessionHistory, question)

    // 3. 调用大模型
    return llmClient.Chat(ctx, messages)
}
```

#### 管理命令

| 命令 | 功能 |
|------|------|
| `/上传知识` | 上传文件加入知识库 |
| `/重建索引` | 重新向量化所有文档 |
| `/知识库列表` | 查看已索引文档 |

#### P4 完成标准

- [ ] 文档上传与解析（MD/TXT/PDF）
- [ ] 向量化与 Qdrant 存储
- [ ] 检索 + 生成问答可用
- [ ] 管理命令可用

#### P4 新增依赖

```
github.com/qdrant/qdrant-go     # 向量库官方 SDK
github.com/ledongthuc/pdf       # PDF 解析（按需）
```

## Docker 部署

### docker-compose.yml（全阶段）

```yaml
services:
  napcat:
    image: mlikiawa/napcat-docker:latest
    environment:
      ACCOUNT: "QQ号"
      TOKEN: "your-token"
    ports:
      - "3001:3001"
    volumes:
      - napcat-data:/app

  bot:
    build: .
    depends_on: [napcat]
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data
    restart: unless-stopped

  # P4 新增
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant-data:/qdrant/storage

volumes:
  napcat-data:
  qdrant-data:
```

### Dockerfile

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

# 运行阶段
FROM alpine:latest
WORKDIR /app
COPY --from=builder /build/bot .
COPY config.yaml .
EXPOSE 8080
ENTRYPOINT ["./bot"]
```

## 各阶段总览

| 阶段 | 核心交付 | 新增依赖 | 代码量估算 |
|------|---------|---------|-----------|
| **P1** | NapCat 对接 + DeepSeek 对话 + 上下文 | zerobot, go-openai, yaml.v3 | ~400 行 |
| **P2** | 命令路由 + 插件系统 + 4 个示例插件 | 无 | ~250 行 |
| **P3** | Agent 工具调用 + 2 个工具 | 无（DeepSeek 原生支持） | ~300 行 |
| **P4** | RAG 知识库 | qdrant-go, pdf | ~350 行 |

每个阶段独立可运行，前一阶段不依赖后一阶段。

## 风险与注意事项

1. **QQ 封号风险**：个人 QQ 号接入有封号可能，建议使用小号。
2. **NapCat 安全**：默认配置存在安全漏洞，部署时必须设置 access_token。
3. **DeepSeek API 限额**：免费额度有限，生产使用需关注用量与费用。
4. **会话历史膨胀**：P1 不做 token 截断，长对话可能超限，后续需加截断策略。
5. **P4 PDF 解析**：`ledongthuc/pdf` 对复杂 PDF 支持有限，可后续按需替换。
