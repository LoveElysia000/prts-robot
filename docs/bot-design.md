# Discord AI 机器人设计文档

## 概述

手写一个基于 Go 的 Discord 聊天机器人，接入 DeepSeek 大模型，支持 AI 角色扮演对话与功能型插件。采用分阶段递进开发，共四个阶段（P1-P4），每阶段独立可运行。

核心设计理念：**角色设定 (Character Skill) 与功能工具 (Function Calling skills) 正交分离**。角色决定"我是谁、怎么说话"，功能工具决定"我能做什么"，两维度独立维护，通过配置交汇。

## 需求摘要

| 维度 | 决策 |
|------|------|
| 平台 | Discord（官方 API，discordgo） |
| 语言 | Go 1.25+，角色生成调用 Python 解析器（子进程） |
| 触发方式 | @机器人回复 + 私聊始终回复，可配置 |
| 上下文记忆 | 全量历史，按频道/DM 隔离，SQLite 持久化存储 |
| 大模型 | DeepSeek V4 Flash（OpenAI 兼容格式） |
| 角色系统 | 按群绑定角色，角色 SKILL.md 作为 system prompt |
| 角色生成 | 从 PRTS Wiki 页面一条命令生成，调用 prts-character-skill 解析器 |
| 功能范围 | AI 角色扮演对话 + 娱乐小工具 + 群管理 + Agent 工具调用 + RAG 知识库 |
| 部署 | Docker + Docker Compose（Bot 单容器 + 可选 Qdrant） |

## 技术栈

| 层次 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.22+ | 主体逻辑，goroutine 并发处理消息 |
| Python 运行时 (仅角色生成) | Python 3.12 | 调用 prts_parser.py 做精确页面解析 |
| 依赖管理 | Go modules + uv | 主项目 go.mod，Python 依赖用 uv |
| 协议 | Discord 官方 API + discordgo | WebSocket 长连接，@/DM 事件 |

| 大模型 SDK | `github.com/sashabaranov/go-openai` | OpenAI 兼容，BaseURL 指向 DeepSeek |
| HTML 解析 | `github.com/PuerkitoBio/goquery` | 角色生成时提取页面正文 |
| 文件监听 | `github.com/fsnotify/fsnotify` | 角色文件热加载 |
| 配置 | `gopkg.in/yaml.v3` | YAML 配置文件 |
| 日志 | `log/slog`（标准库） | 结构化日志 |
| 数据库 | SQLite + `modernc.org/sqlite` | 会话历史持久化（纯 Go，嵌入式） |
| 向量库 (P4) | Qdrant + `github.com/qdrant/qdrant-go` | 官方 Go SDK，Docker 部署 |
| Embedding (P4) | 智谱 `embedding-3` | DeepSeek 不提供公开 Embedding API |
| 文档解析 (P4) | `github.com/ledongthuc/pdf` + 标准库 | PDF/MD/TXT 解析 |
| 角色生成 (Python) | `prts_parser.py` + `character_skill_writer.py` | 复用 prts-character-skill 工具 |
| 容器化 | Docker + Docker Compose | Bot 单容器 (+ Qdrant for P4) |

### 选型理由

- **Go + Python 解析器**：主体 Go（单二进制、并发好），角色生成调用用户已有的 Python 解析器保证质量。
- **Discord 官方 API + discordgo：零封号风险，WebSocket 长连接，@消息 + DM 支持。
- **go-openai**：DeepSeek 兼容 OpenAI 格式，自定义 BaseURL 即可接入。
- **SQLite**：嵌入式数据库，零运维，WAL 模式保证并发安全。会话数据全量保存，发给 DeepSeek 的只取最近 N 轮窗口。
- **Qdrant 而非 ChromaDB**：ChromaDB 无 Go SDK，Qdrant 有官方 Go SDK 且支持 Docker。
- **智谱 Embedding**：DeepSeek 不提供公开 Embedding API，智谱 embedding-3 是国内性价比最高的替代。
- **复用 prts-character-skill**：角色生成的质量保障来自精确的 Python 解析器，Go 通过子进程调用。

## 整体架构

```
┌─────────────────────────────────────────────────────┐
│  Discord 客户端                                           │
└──────────┬──────────────────────────────────────────┘
┌──────────▼──────────────────────────────────────────┐
│  Discord Gateway                                         │
│  WebSocket 事件推送 + Bot 回复消息         │
└──────────┬──────────────────────────────────────────┘
           │ WebSocket → Bot / Bot → API → Discord
┌──────────▼──────────────────────────────────────────┐
│  Bot 核心 (Go + Python)                              │
│                                                      │
│  ┌─────────────┐   ┌──────────────┐                 │
│  │ 消息接收/解析 │──▶│ 消息处理管线  │                 │
│  └─────────────┘   └──────┬───────┘                 │
│                           │                          │
│     ┌─────────────────────┼──────────────────┐       │
│     ▼                     ▼                  ▼       │
│  ┌────────────┐   ┌──────────────┐   ┌──────────┐  │
│  │ 命令路由    │   │ AI 对话管线   │   │ 角色生成器│  │
│  │ (P2/P3)    │   │ (P1/P3/P4)   │   │ (P2)     │  │
│  └─────┬──────┘   └──────┬───────┘   └────┬─────┘  │
│        │                 │                 │         │
│        │         ┌───────▼────────┐       │         │
│        │         │ Persona Manager│◄──────┘         │
│        │         │ 加载 SKILL.md  │                 │
│        │         │ 按群绑定角色    │                 │
│        │         └───────┬────────┘                 │
│        │                 │                          │
│        ▼                 ▼                          │
│  ┌────────────┐   ┌──────────────┐                 │
│  │ 插件/工具   │   │ DeepSeek     │                 │
│  │ + 命令映射  │   │ + Agent 工具 │                 │
│  └────────────┘   │ + RAG (P4)   │                 │
│                   └──────────────┘                 │
└─────────────────────────────────────────────────────┘
```

### 消息流转（完整链路）

```
收到消息
  │
  ├─ 以 "/" 开头？──→ 命令路由
  │   ├─ /生成角色 {名字} {Wiki URL}    角色生成 (Go→Python→DeepSeek)
  │   ├─ /角色校正 {名字} {修正内容}      局部校正
  │   ├─ /角色 列表/切换/信息/重载        角色管理
  │   ├─ /天气 {城市}                    工具直接调用 (P3)
  │   ├─ /今日运势, /禁言 ...             P2 功能插件
  │   └─ 未匹配 → 提示无此命令
  │
  └─ 普通消息 ──→ AI 对话管线
                   │
                   ▼
              1. SQLite 写入用户消息 (INSERT)
              2. Persona Manager 查当前群绑定角色
              3. 加载 SKILL.md → system prompt
              4. SQLite 读取最近 20 轮历史 (SELECT ... LIMIT 40)
              5. 取角色 skills 列表 → 过滤工具
              6. 组装请求 (prompt + 历史 + 用户消息 + tools)
              7. DeepSeek → tool_call 或 纯文本回复
              8. SQLite 写入 AI 回复 (INSERT)
```

## 两个"skill"概念（正交分离）

| 概念 | 含义 | 产物 | 作用 |
|------|------|------|------|
| **Character Skill** | 角色设定文件 | `SKILL.md` | system prompt，决定"我是谁、怎么说话" |
| **Function skills** | 功能工具 | `tools/*.go` | Function Calling 参数，决定"能做什么" |

两者在 `personas.yaml` 的 `skills` 字段交汇——角色声明拥有哪些工具。

## 项目目录结构（全阶段）

```
robot/
├── cmd/
│   └── bot/
│       └── main.go                 # 入口
├── internal/
│   ├── core/
│   │   ├── qqapi.go（已废弃，Discord 用 discordgo）                # Discord Gateway 客户端（Token + 发消息）
│   │   └── config.go               # 配置加载
│   ├── message/
│   │   ├── types.go                 # 消息结构体
│   │   └── handler.go               # 消息处理管线（触发判断 + 分流）
│   ├── session/
│   │   └── manager.go              # 会话管理（SQLite 全量存储，窗口读取）
│   ├── persona/                    # 角色系统 (P2)
│   │   ├── manager.go              # 加载、绑定、热加载、文件监听
│   │   ├── card.go                 # 角色卡结构定义
│   │   ├── corrector.go            # AI 局部校正
│   │   └── generator/              # 交互式生成器
│   │       ├── generator.go        # 流程编排（一条命令生成）
│   │       ├── fetcher.go          # 抓取 Wiki 页面 + goquery 提取正文
│   │       ├── parser_bridge.go    # 调用 Python prts_parser.py
│   │       └── writer.go           # 调用 Python character_skill_writer.py
│   ├── llm/
│   │   ├── client.go               # DeepSeek 客户端 (P1)
│   │   ├── agent.go                # Function Calling 调度 (P3)
│   │   └── tools/                  # Function 工具目录 (P3)
│   │       ├── registry.go         # 注册表 + 按 skills 过滤
│   │       ├── web_search.go
│   │       └── weather.go
│   ├── plugins/                    # 命令插件 (P2)
│   │   ├── registry.go             # 插件注册
│   │   ├── plugin.go               # 插件接口
│   │   ├── entertainment/
│   │   │   ├── fortune.go
│   │   │   └── random_image.go
│   │   └── groupadmin/
│   │       ├── welcome.go
│   │       └── mute.go
│   └── rag/                        # RAG 知识库 (P4)
│       ├── indexer.go
│       ├── retriever.go
│       └── store.go
├── tools/                          # Python 工具 (从 prts-character-skill 拷贝 2 个脚本)
│   ├── prts_parser.py              # 页面精确解析器
│   ├── character_skill_writer.py   # SKILL.md 拼装
│   └── requirements.txt            # bs4, lxml
├── prompts/                        # 角色生成规则 (从 prts-character-skill 拷贝 7 个文件)
│   ├── persona_builder.md
│   ├── lore_builder.md
│   ├── relationship_builder.md
│   ├── custom_builder.md
│   ├── correction_handler.md
│   ├── merger.md
│   └── custom_global.md            # writer.py 引用的全局模板
├── data/
│   ├── bot.db                       # SQLite 数据库（会话历史、角色绑定）
│   ├── personas/                    # 角色设定文件 (写操作，需持久化挂载)
│   │   ├── lin/
│   │   │   ├── SKILL.md
│   │   │   ├── persona.md
│   │   │   ├── lore.md
│   │   │   ├── relationship.md
│   │   │   ├── custom.md
│   │   │   └── meta.json
│   │   └── chen/
│   ├── personas.yaml               # 角色绑定配置（热加载）
│   └── qdrant/                     # 向量库 (P4)
├── config.yaml
├── go.mod
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 配置文件设计

### config.yaml

```yaml
# config.yaml
qq:
  webhook_port: 8080

deepseek:
  api_key: "sk-xxx"
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"              # deepseek-chat 于 2026/07/24 弃用

trigger:
  mode: "hybrid"          # all | at | hybrid
  command_prefix: "/"

database:
  path: "./data/bot.db"               # SQLite 数据库路径

persona:
  config_path: "./data/personas.yaml"
  hot_reload: true

# P4
qdrant:
  url: "http://qdrant:6333"
  collection: "knowledge"

embedding:
  provider: "zhipu"          # DeepSeek 不提供公开 Embedding API
  api_key: "xxx"
  model: "embedding-3"
```

### data/personas.yaml

```yaml
# data/personas.yaml
personas:
  lin:
    name: "林"
    slug: "lin"
    skill_dir: "data/personas/lin"
    skills: [web_search, weather]    # 角色拥有的 Function 工具

  chen:
    name: "陈"
    slug: "chen"
    skill_dir: "data/personas/chen"
    skills: [web_search]

  default:
    name: "助手"
    skill_dir: ""                    # 空 = 用 config 的 default_system_prompt
    skills: []

bindings:
  "group_12345": "lin"
  "group_67890": "chen"
  "private_*": "default"
```

## 角色系统设计 (P2)

### Character Skill 结构

角色文件来自 `prts-character-skill` 工具生成的标准产物：

```
data/personas/{slug}/
├── SKILL.md           # 完整角色 Skill，作为 system prompt
├── persona.md         # 人格、说话方式、情绪机制、互动边界
├── lore.md            # 设定事实、经历、关系、能力
├── relationship.md    # 关系身份
├── custom.md          # 语言纹理、长期陪伴感
└── meta.json          # 元数据
```

### Persona Manager

```go
// internal/persona/manager.go
type PersonaManager struct {
    mu       sync.RWMutex
    personas map[string]*Persona  // slug -> 角色
    bindings map[string]string    // sessionKey -> slug
    configPath string
}

type Persona struct {
    Name     string
    Slug     string
    SkillDir string
    Skills   []string            // 拥有的 Function 工具名列表
    Prompt   string              // 加载后的 SKILL.md 内容
}

// 加载 SKILL.md
func (p *Persona) LoadPrompt() (string, error)

// 查当前群绑定的角色（含回退逻辑）
func (pm *PersonaManager) GetForSession(sessionKey string) (*Persona, error) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()

    slug, ok := pm.bindings[sessionKey]
    if !ok {
        slug = "default"  // 未绑定时回退到 default 角色
    }
    persona, ok := pm.personas[slug]
    if !ok {
        return nil, fmt.Errorf("角色不存在: %s", slug)
    }
    return persona, nil
}
```

### 命令

| 命令 | 功能 |
|------|------|
| `/生成角色 {名字} {Wiki URL}` | 一条命令生成 + 自动绑定 |
| `/角色校正 {名字} {修正内容}` | 局部 AI 校正 |
| `/角色 列表` | 查看所有角色 |
| `/角色 切换 {名字}` | 当前群换角色 |
| `/角色 信息 {名字}` | 查看角色详情 |
| `/角色 重载` | 热加载文件 |

### 文件热加载

```go
// internal/persona/manager.go
func (pm *PersonaManager) Watch(ctx context.Context) {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(filepath.Dir(pm.configPath))
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
                pm.reload()
            }
        case <-ctx.Done():
            return
        }
    }
}
```

## 角色生成器设计 (P2)

### 概述

从 PRTS Wiki 页面一条命令生成角色，复用 `prts-character-skill` 的 Python 精确解析器保障质量。

```
/生成角色 林 https://prts.wiki/w/林

内部流程（用户无感，一条命令完成）:
  1. Go (net/http) 抓取页面 HTML
  2. Go (exec) 调用 prts_parser.py → 结构化 JSON (质量保障)
  3. JSON + prompts 规则 → DeepSeek 并行生成 persona/lore/relationship/custom 四层
  4. Go (exec) 调用 character_skill_writer.py → 拼装 SKILL.md
  5. 保存到 data/personas/{slug}/ + 写入 personas.yaml 绑定
```

### 组件

```
internal/persona/generator/
├── generator.go       # 流程编排（一条命令生成）
├── fetcher.go         # 抓取 Wiki 页面
├── parser_bridge.go   # 调用 Python prts_parser.py (子进程)
└── writer.go          # 调用 Python character_skill_writer.py (子进程)
```

### 核心代码

```go
// internal/persona/generator/generator.go
type Generator struct {
    llm        *llm.DeepSeekClient
    prompts    map[string]string  // 加载 prompts/ 目录的规则文件（镜像内 /app/prompts/）
    pythonDir  string             // tools/ 目录（镜像内固定为 ./tools/）
    outputDir  string             // data/personas/
    manager    *PersonaManager
}

type GenerateRequest struct {
    Slug    string  // 角色标识，如 "lin"
    Name    string  // 角色名，如 "林"
    WikiURL string  // PRTS Wiki URL
    GroupID string  // 要绑定的群
}

func (g *Generator) Generate(ctx context.Context, req GenerateRequest) (*Persona, error) {
    // 1. Go 抓取页面
    htmlPath, err := g.fetcher.SaveHTML(ctx, req.WikiURL, req.Slug)

    // 2. 调用 Python 精确解析 (质量保障)
    profileJSON, err := g.callParser(ctx, htmlPath, req.Slug)

    // 3. 四层并行生成 (goroutine)
    var (
        persona, lore, relationship, custom string
        errs [4]error
    )
    var wg sync.WaitGroup
    wg.Add(4)
    go func() { defer wg.Done(); persona, errs[0] = g.generateLayer(ctx, "persona_builder", profileJSON, req.Name) }()
    go func() { defer wg.Done(); lore, errs[1] = g.generateLayer(ctx, "lore_builder", profileJSON, req.Name) }()
    go func() { defer wg.Done(); relationship, errs[2] = g.generateLayer(ctx, "relationship_builder", profileJSON, req.Name) }()
    go func() { defer wg.Done(); custom, errs[3] = g.generateLayer(ctx, "custom_builder", profileJSON, req.Name) }()
    wg.Wait()

    for i, err := range errs {
        if err != nil { return nil, fmt.Errorf("第%d层生成失败: %w", i+1, err) }
    }

    // 4. 保存分层文件
    dir := g.outputDir + "/" + req.Slug
    os.MkdirAll(dir, 0755)
    os.WriteFile(dir+"/persona.md", []byte(persona), 0644)
    os.WriteFile(dir+"/lore.md", []byte(lore), 0644)
    os.WriteFile(dir+"/relationship.md", []byte(relationship), 0644)
    os.WriteFile(dir+"/custom.md", []byte(custom), 0644)

    // 5. 调用 Python writer 拼装 SKILL.md
    g.callWriter(req.Slug, req.Name, req.WikiURL, dir)

    // 6. 自动绑定
    g.manager.Bind("group_"+req.GroupID, req.Slug)

    return g.manager.Get(req.Slug)
}

func (g *Generator) generateLayer(ctx context.Context, ruleName string, profileJSON string, name string) (string, error) {
    rule := g.prompts[ruleName]
    messages := []openai.ChatCompletionMessage{
        {Role: openai.ChatMessageRoleSystem, Content: rule},
        {Role: openai.ChatMessageRoleUser, Content: fmt.Sprintf(
            "角色名: %s\n\n解析结果:\n%s", name, profileJSON)},
    }
    return g.llm.Chat(ctx, messages)
}
```

```go
// internal/persona/generator/parser_bridge.go
func (g *Generator) callParser(ctx context.Context, htmlPath string, slug string) (string, error) {
    outputPath := filepath.Join(os.TempDir(), "prts_"+slug+".json")
    cmd := exec.CommandContext(ctx, "python3",
        filepath.Join(g.pythonDir, "prts_parser.py"),
        "--input", htmlPath,
        "--output", outputPath,
    )
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("解析失败: %w", err)
    }
    data, err := os.ReadFile(outputPath)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

### 质量保障

```
三层兜底:
  1. Python 精确解析 (prts_parser.py)   ← 替代通用 HTML→文本，和原工具质量一致
  2. 四层独立生成 (goroutine 并行)       ← 每层用专用 prompt 规则，非一次性吐出
  3. 事后校正 (/角色校正)                ← 用 correction_handler 规则局部修正
```

### 依赖

```
github.com/PuerkitoBio/goquery   # HTML 提取
github.com/fsnotify/fsnotify     # 文件监听
# Python: bs4 + lxml (tools/requirements.txt)
```

---

## 阶段设计

### P1：核心对话

**目标**：机器人上线，能进行带上下文的 AI 对话。

#### 组件职责

| 文件 | 职责 |
|------|------|
| `cmd/bot/main.go` | 入口，加载配置，启动 Discord Session |
| `internal/core/bot.go` | WebSocket 消息处理 + 消息处理管线 |
| `internal/core/qqapi.go（已废弃，Discord 用 discordgo）` | Discord Gateway 客户端（Token + 群消息） |
| `internal/core/config.go` | 读取 config.yaml，结构体映射 |
| `internal/message/types.go` | 消息结构体与辅助方法 |
| `internal/message/handler.go` | 判断触发条件，分流到 AI 管线 |
| `internal/session/manager.go` | 按 group_id 维护全量历史，SQLite 持久化 |
| `internal/llm/client.go` | 封装 DeepSeek API 调用（go-openai） |

#### 触发逻辑

```go
func ShouldReply(msg *Message, cfg *Config) bool {
    if msg.IsPrivate {
        return true
    }
    switch cfg.Trigger.Mode {
    case "all":     return true
    case "at":      return msg.IsAtBot
    case "hybrid":  return msg.IsAtBot  // 私聊全回 + 群@回
    }
    return false
}
```

#### 会话管理

- 会话 key：`group_{group_id}` / `private_{user_id}`
- 全量存储于 SQLite `data/bot.db`，每条消息 INSERT 入库
- 发给 DeepSeek 时只取最近 20 轮（`ORDER BY id DESC LIMIT 40`），受上下文窗口限制
- SQLite WAL 模式保证并发安全，无需额外服务
- P1 不做自动清理旧数据

#### SQLite 表结构

```sql
CREATE TABLE messages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_key TEXT NOT NULL,       -- "group_12345" 或 "private_67890"
    role        TEXT NOT NULL,       -- "user" / "assistant"
    content     TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_messages_session ON messages(session_key, id);
```

```go
// internal/session/manager.go
type SessionManager struct {
    db *sql.DB
}

func (sm *SessionManager) Append(sessionKey string, msg Message) error {
    _, err := sm.db.Exec(
        `INSERT INTO messages (session_key, role, content) VALUES (?, ?, ?)`,
        sessionKey, msg.Role, msg.Content,
    )
    return err
}

func (sm *SessionManager) GetRecent(sessionKey string, rounds int) ([]Message, error) {
    rows, err := sm.db.Query(
        `SELECT role, content FROM messages 
         WHERE session_key = ? 
         ORDER BY id DESC LIMIT ?`, sessionKey, rounds*2,
    )
    // 倒序取出 → 翻转后返回
    ...
}
```

#### P1 完成标准

- [ ] 群聊 @机器人 正常对话，带上下文
- [ ] 会话历史 SQLite 持久化，重启不丢
- [ ] Docker Compose 一键启动

#### P1 依赖

```
github.com/sashabaranov/go-openai
modernc.org/sqlite
gopkg.in/yaml.v3
# net/http, crypto/ed25519 为标准库
```

---

### P2：角色系统 + 命令插件

**目标**：角色系统上线 + 命令路由框架 + 插件系统。

#### 角色系统

- PersonaManager：加载 SKILL.md，按群绑定，监听文件变化热加载
- 角色生成器：一条命令从 Wiki 页面生成角色，调用 Python 解析器保障质量
- 角色校正器：AI 局部修正

#### 命令路由

- 消息以 `command_prefix`（默认 `/`）开头 → 命令路由
- 否则 → AI 对话管线（注入角色 system prompt）

#### 角色命令

| 命令 | 功能 |
|------|------|
| `/生成角色 {名字} {Wiki URL}` | 一条命令生成并绑定 |
| `/角色校正 {名字} {内容}` | 局部修正 |
| `/角色 列表` | 查看所有角色 |
| `/角色 切换 {名字}` | 切换绑定 |
| `/角色 信息 {名字}` | 查看详情 |
| `/角色 重载` | 热加载 |

#### 功能插件

| 插件 | 命令 | 功能 |
|------|------|------|
| 今日运势 | `/今日运势` | 随机运势 |
| 随机图片 | `/随机图片` | 随机图 API |
| 入群欢迎 | （事件触发） | 自动欢迎语 |

#### P2 完成标准

- [ ] 角色系统可用，按群绑定角色，对话带人设
- [ ] 角色生成器可用，一条命令生成并绑定
- [ ] 角色热加载可用
- [ ] `/help` 列出所有命令
- [ ] 2 个娱乐 + 2 个群管插件可用

#### P2 新增依赖

```
github.com/PuerkitoBio/goquery
github.com/fsnotify/fsnotify
# Python: bs4, lxml
```

---

### P3：AI Agent 工具调用

**目标**：Function Calling 集成，角色能按 skills 声明调用工具。

#### 设计要点

- 工具放 `internal/llm/tools/` 目录自动注册
- 角色通过 `personas.yaml` 的 `skills` 字段声明拥有哪些工具
- 调 DeepSeek 时按角色 skills 过滤工具列表
- 工具同时支持 AI 自主调用和 `/命令` 直接调用

#### 工具定义

```go
// internal/llm/tools/weather.go
func init() {
    Register("weather", RegisteredTool{
        Schema: openai.Tool{
            Type: openai.ToolTypeFunction,
            Function: &openai.FunctionDefinition{
                Name: "get_weather",
                Description: "查询指定城市天气",
                Parameters: { /* JSON Schema */ },
            },
        },
        Execute: executeWeather,
    })
}
```

#### P3 完成标准

- [ ] DeepSeek Function Calling 集成
- [ ] 工具自动注册
- [ ] 按角色 skills 过滤工具
- [ ] 2 个工具：网页搜索 + 天气查询
- [ ] 工具命令映射（/天气 北京 直接调用）

---

### P4：RAG 知识库

**目标**：文档上传、向量化、检索增强问答。

#### Embedding 选型

DeepSeek 不提供公开 Embedding API，使用智谱 `embedding-3` 替代。

#### 架构

```
上传文档 → 解析分块 → 智谱 Embedding → Qdrant

提问 → 向量化 → Qdrant 检索 Top-K → 注入 prompt → DeepSeek 生成
```

#### P4 完成标准

- [ ] 文档上传与解析（MD/TXT/PDF）
- [ ] 向量化与 Qdrant 存储
- [ ] 检索 + 生成问答可用
- [ ] 管理命令可用

#### P4 新增依赖

```
github.com/qdrant/qdrant-go
github.com/ledongthuc/pdf
```

---

## Docker 部署

### docker-compose.yml

```yaml
services:
  bot:
    build: .
    ports:
      - "8080:8080"              # 
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data         # 包含 bot.db + personas/
    restart: unless-stopped

  # P4 新增
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant-data:/qdrant/storage

volumes:
  qdrant-data:
```


2. 



### Dockerfile

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

# 运行阶段 (含 Python 用于角色生成)
FROM python:3.12-alpine
WORKDIR /app

# 基础依赖
RUN apk add --no-cache ca-certificates tzdata

# 复制 Go 二进制
COPY --from=builder /build/bot .
COPY config.yaml .

# 确保 data 目录存在（bot.db 将由程序自动创建）
RUN mkdir -p /app/data

# Python 依赖 (角色生成)
COPY tools/ ./tools/
COPY prompts/ ./prompts/            # 放在 /app/prompts/，不在 data 挂载点下
RUN pip install --no-cache-dir beautifulsoup4 lxml

EXPOSE 8080
ENTRYPOINT ["./bot"]
```

## 各阶段总览

| 阶段 | 核心交付 | 新增依赖 | 代码量估算 |
|------|---------|---------|-----------|
| **P1** | Discord + DeepSeek 对话 + SQLite 会话 | go-openai, sqlite, yaml.v3 | ~550 行 |
| **P2** | 角色系统 + 生成器 + 命令路由 + 插件 | goquery, fsnotify, Python解析器 | ~500 行 |
| **P3** | Agent 工具 + skills 过滤 + 命令映射 | 无 | ~300 行 |
| **P4** | RAG 知识库 | qdrant-go, pdf | ~350 行 |

## 风险与注意事项

3. **
4. **DeepSeek 模型名**：`deepseek-chat` 于 2026/07/24 弃用，使用 `deepseek-v4-flash`。
5. **DeepSeek 无 Embedding**：P4 使用智谱 embedding-3 替代。
6. **SQLite 数据膨胀**：全量存储意味着 `data/bot.db` 会持续增长。个人使用场景下增长缓慢（数月才几十 MB），P1 不做自动清理。
7. **角色生成耗时**：涉及 Python 调用 + 四次 DeepSeek 请求，预计 10-30 秒，需做好超时处理和进度反馈。
8. **
