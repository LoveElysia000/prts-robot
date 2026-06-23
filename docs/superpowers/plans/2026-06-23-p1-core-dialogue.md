# P1: 核心对话 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 机器人上线，能进行带上下文的 AI 对话（私聊 + 群@回复），SQLite 持久化会话历史，Docker Compose 一键启动。

**Architecture:** ZeroBot 作为 OneBot 客户端监听反向 WS，收到消息后经触发判断分流到 AI 管线，AI 管线从 SQLite 取最近 20 轮历史 + SKILL.md → DeepSeek API → 回复。

**Tech Stack:** Go 1.22+, ZeroBot, go-openai, modernc.org/sqlite, yaml.v3

---

## 文件结构

```
robot/
├── cmd/bot/main.go                  # 入口
├── internal/
│   ├── core/
│   │   ├── config.go                # 配置加载
│   │   └── bot.go                   # ZeroBot 初始化 + 事件注册
│   ├── message/
│   │   ├── types.go                 # 消息结构体
│   │   ├── handler.go               # 触发判断 + 消息处理管线
│   │   └── sender.go                # 回复封装与发送
│   ├── session/
│   │   └── manager.go               # SQLite 会话管理
│   └── llm/
│       └── client.go                # DeepSeek 客户端
├── config.yaml                      # 配置文件
├── go.mod / go.sum
├── Dockerfile
└── docker-compose.yml
```

---

### Task 1: 项目脚手架

**Files:**
- Create: `go.mod`
- Create: `config.yaml`
- Create: `cmd/bot/main.go`（占位）

- [ ] **Step 1: 初始化 Go module**

```bash
cd /Users/Ein/project2/robot && go mod init github.com/loveelysia000/robot
```
Run: `cat go.mod`
Expected: 显示 `module github.com/loveelysia000/robot` 和 `go 1.22`

- [ ] **Step 2: 创建 config.yaml 模板**

```yaml
# config.yaml
napcat:
  port: 8080
  access_token: "change-me"

deepseek:
  api_key: "sk-change-me"
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  default_system_prompt: "你是一个友好的QQ群助手"

trigger:
  mode: "hybrid"
  command_prefix: "/"

database:
  path: "./data/bot.db"
```

- [ ] **Step 3: 创建占位 main.go**

```go
// cmd/bot/main.go
package main

import "fmt"

func main() {
	fmt.Println("robot starting...")
}
```

Run: `go run ./cmd/bot`
Expected: `robot starting...`

- [ ] **Step 4: 创建目录结构**

```bash
mkdir -p internal/core internal/message internal/session internal/llm
mkdir -p data
```

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "chore: project scaffolding"
```

---

### Task 2: 配置加载

**Files:**
- Create: `internal/core/config.go`
- Create: `internal/core/config_test.go`

- [ ] **Step 1: 写 config_test.go**

```go
// internal/core/config_test.go
package core

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := `
napcat:
  port: 8080
  access_token: "test-token"
deepseek:
  api_key: "sk-test"
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  default_system_prompt: "test prompt"
trigger:
  mode: "hybrid"
  command_prefix: "/"
database:
  path: "./data/bot.db"
`
	f, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.NapCat.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.NapCat.Port)
	}
	if cfg.DeepSeek.Model != "deepseek-v4-flash" {
		t.Errorf("expected model deepseek-v4-flash, got %s", cfg.DeepSeek.Model)
	}
	if cfg.Trigger.Mode != "hybrid" {
		t.Errorf("expected mode hybrid, got %s", cfg.Trigger.Mode)
	}
	if cfg.Database.Path != "./data/bot.db" {
		t.Errorf("expected db path ./data/bot.db, got %s", cfg.Database.Path)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/core/ -v -run TestLoadConfig
```
Expected: FAIL — `LoadConfig` 未定义

- [ ] **Step 3: 写 config.go 最小实现**

```go
// internal/core/config.go
package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	NapCat   NapCatConfig   `yaml:"napcat"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

type NapCatConfig struct {
	Port        int    `yaml:"port"`
	AccessToken string `yaml:"access_token"`
}

type DeepSeekConfig struct {
	APIKey               string `yaml:"api_key"`
	BaseURL              string `yaml:"base_url"`
	Model                string `yaml:"model"`
	DefaultSystemPrompt  string `yaml:"default_system_prompt"`
}

type TriggerConfig struct {
	Mode          string `yaml:"mode"`
	CommandPrefix string `yaml:"command_prefix"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
```

- [ ] **Step 4: 安装依赖 + 运行测试**

```bash
go get gopkg.in/yaml.v3
go test ./internal/core/ -v -run TestLoadConfig
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/ go.mod go.sum
git commit -m "feat: config loading with yaml.v3"
```

---

### Task 3: 消息类型定义

**Files:**
- Create: `internal/message/types.go`
- Create: `internal/message/types_test.go`

- [ ] **Step 1: 写 types_test.go**

```go
// internal/message/types_test.go
package message

import "testing"

func TestIsCommand(t *testing.T) {
	msg := &Message{Text: "/help"}
	if !msg.IsCommand("/") {
		t.Error("expected /help to be command")
	}

	msg2 := &Message{Text: "你好"}
	if msg2.IsCommand("/") {
		t.Error("expected 你好 not to be command")
	}
}

func TestSessionKey(t *testing.T) {
	msgGroup := &Message{GroupID: "12345"}
	if key := msgGroup.SessionKey(); key != "group_12345" {
		t.Errorf("expected group_12345, got %s", key)
	}

	msgPrivate := &Message{UserID: "67890"}
	if key := msgPrivate.SessionKey(); key != "private_67890" {
		t.Errorf("expected private_67890, got %s", key)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/message/ -v
```
Expected: FAIL — `Message` 未定义

- [ ] **Step 3: 写 types.go**

```go
// internal/message/types.go
package message

import "strings"

type Message struct {
	GroupID string
	UserID  string
	Text    string
	IsAtBot bool
}

func (m *Message) IsCommand(prefix string) bool {
	return strings.HasPrefix(m.Text, prefix)
}

func (m *Message) SessionKey() string {
	if m.GroupID != "" {
		return "group_" + m.GroupID
	}
	return "private_" + m.UserID
}

func (m *Message) IsPrivate() bool {
	return m.GroupID == ""
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/message/ -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/message/ && git commit -m "feat: message types with command check and session key"
```

---

### Task 4: SQLite 会话管理

**Files:**
- Create: `internal/session/manager.go`
- Create: `internal/session/manager_test.go`

- [ ] **Step 1: 安装 SQLite 依赖**

```bash
go get modernc.org/sqlite
```

- [ ] **Step 2: 写 manager_test.go**

```go
// internal/session/manager_test.go
package session

import (
	"database/sql"
	"os"
	"testing"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	f, _ := os.CreateTemp("", "test-*.db")
	f.Close()
	db, err := sql.Open("sqlite", f.Name())
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	return db
}

func TestAppendAndGetRecent(t *testing.T) {
	db := setupDB(t)
	mgr, err := NewManager(db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.Append("group_123", Message{Role: "user", Content: "你好"})
	mgr.Append("group_123", Message{Role: "assistant", Content: "你好！"})
	mgr.Append("group_123", Message{Role: "user", Content: "天气？"})
	mgr.Append("group_123", Message{Role: "assistant", Content: "28°C"})

	recent, err := mgr.GetRecent("group_123", 2)
	if err != nil {
		t.Fatalf("GetRecent failed: %v", err)
	}

	if len(recent) != 4 {
		t.Fatalf("expected 4 messages (2 rounds), got %d", len(recent))
	}
	if recent[0].Role != "user" || recent[0].Content != "你好" {
		t.Errorf("wrong first message: %+v", recent[0])
	}
	if recent[3].Role != "assistant" || recent[3].Content != "28°C" {
		t.Errorf("wrong last message: %+v", recent[3])
	}
}

func TestGetRecentEmpty(t *testing.T) {
	db := setupDB(t)
	mgr, _ := NewManager(db)

	recent, err := mgr.GetRecent("group_nonexist", 10)
	if err != nil {
		t.Fatalf("GetRecent failed: %v", err)
	}
	if len(recent) != 0 {
		t.Errorf("expected 0 messages, got %d", len(recent))
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

```bash
go test ./internal/session/ -v
```
Expected: FAIL — `NewManager` 未定义

- [ ] **Step 4: 写 manager.go**

```go
// internal/session/manager.go
package session

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Message struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) (*Manager, error) {
	query := `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_key TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_key, id);`

	if _, err := db.Exec(query); err != nil {
		return nil, err
	}

	return &Manager{db: db}, nil
}

func (m *Manager) Append(sessionKey string, msg Message) error {
	_, err := m.db.Exec(
		`INSERT INTO messages (session_key, role, content) VALUES (?, ?, ?)`,
		sessionKey, msg.Role, msg.Content,
	)
	return err
}

func (m *Manager) GetRecent(sessionKey string, rounds int) ([]Message, error) {
	rows, err := m.db.Query(
		`SELECT role, content, created_at FROM messages
		 WHERE session_key = ?
		 ORDER BY id DESC LIMIT ?`,
		sessionKey, rounds*2,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	// 倒序取出，翻转回正序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/session/ -v
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/session/ go.mod go.sum && git commit -m "feat: SQLite session manager with append and window read"
```

---

### Task 5: DeepSeek 客户端

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/client_test.go`

- [ ] **Step 1: 安装 go-openai**

```bash
go get github.com/sashabaranov/go-openai
```

- [ ] **Step 2: 写 client_test.go**

```go
// internal/llm/client_test.go
package llm

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestBuildMessages(t *testing.T) {
	client := &Client{}
	sysPrompt := "你是一个助手"
	sessionMsgs := []ChatMessage{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好！"},
	}
	userText := "天气？"

	messages := client.BuildMessages(sysPrompt, sessionMsgs, userText, nil)

	if len(messages) != 4 {
		t.Fatalf("expected 4 messages (system + 2 history + 1 user), got %d", len(messages))
	}
	if messages[0].Role != openai.ChatMessageRoleSystem {
		t.Error("first message should be system")
	}
	if messages[1].Role != openai.ChatMessageRoleUser || messages[1].Content != "你好" {
		t.Error("second message should be user '你好'")
	}
	if messages[3].Role != openai.ChatMessageRoleUser || messages[3].Content != "天气？" {
		t.Error("last message should be user '天气？'")
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

```bash
go test ./internal/llm/ -v -run TestBuildMessages
```
Expected: FAIL

- [ ] **Step 4: 写 client.go**

```go
// internal/llm/client.go
package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type ChatMessage struct {
	Role    string
	Content string
}

type Client struct {
	api  *openai.Client
	model string
}

func NewClient(cfg DeepSeekConfig) *Client {
	api := openai.NewClientWithConfig(openai.DefaultConfig(cfg.APIKey))
	api.BaseURL = cfg.BaseURL
	return &Client{
		api:   api,
		model: cfg.Model,
	}
}

type DeepSeekConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func (c *Client) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("deepseek api error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from deepseek")
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *Client) BuildMessages(systemPrompt string, sessionMsgs []ChatMessage, userText string, tools []openai.Tool) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, 1+len(sessionMsgs)+1)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	for _, m := range sessionMsgs {
		role := openai.ChatMessageRoleUser
		if m.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userText,
	})
	return messages
}
```

- [ ] **Step 5: 运行测试**

```bash
go test ./internal/llm/ -v -run TestBuildMessages
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/llm/ go.mod go.sum && git commit -m "feat: deepseek client with message builder"
```

---

### Task 6: 消息处理管线（触发判断 + 分流）

**Files:**
- Create: `internal/message/handler.go`
- Create: `internal/message/handler_test.go`

- [ ] **Step 1: 写 handler_test.go**

```go
// internal/message/handler_test.go
package message

import (
	"testing"
)

func TestShouldReply(t *testing.T) {
	cfg := TriggerConfig{
		Mode:          "hybrid",
		CommandPrefix: "/",
	}

	// 私聊始终回复
	if !ShouldReply(&Message{UserID: "1", Text: "你好"}, cfg) {
		t.Error("private msg should always reply")
	}

	// hybrid 模式：群@才回
	if !ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: true}, cfg) {
		t.Error("hybrid mode: at msg should reply")
	}
	if ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfg) {
		t.Error("hybrid mode: non-at msg should not reply")
	}

	// all 模式
	cfgAll := TriggerConfig{Mode: "all", CommandPrefix: "/"}
	if !ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfgAll) {
		t.Error("all mode: should reply")
	}

	// at 模式
	cfgAt := TriggerConfig{Mode: "at", CommandPrefix: "/"}
	if ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfgAt) {
		t.Error("at mode: non-at should not reply")
	}
}

type TriggerConfig struct {
	Mode          string
	CommandPrefix string
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/message/ -v -run TestShouldReply
```
Expected: FAIL

- [ ] **Step 3: 写 handler.go**

```go
// internal/message/handler.go
package message

func ShouldReply(msg *Message, cfg TriggerConfig) bool {
	if msg.IsPrivate() {
		return true
	}
	switch cfg.Mode {
	case "all":
		return true
	case "at":
		return msg.IsAtBot
	case "hybrid":
		return msg.IsAtBot
	}
	return false
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/message/ -v -run TestShouldReply
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/message/handler.go internal/message/handler_test.go && git commit -m "feat: trigger logic with shouldReply"
```

---

### Task 7: 消息发送器

**Files:**
- Create: `internal/message/sender.go`

- [ ] **Step 1: 写 sender.go**

```go
// internal/message/sender.go
package message

import (
	"fmt"

	zero "github.com/wdvxdr1123/zerobot"
)

type Sender struct {
	ctx *zero.Ctx
}

func NewSender(ctx *zero.Ctx) *Sender {
	return &Sender{ctx: ctx}
}

func (s *Sender) SendGroup(groupID, text string) error {
	s.ctx.SendGroupMessage(groupID, text)
	return nil
}

func (s *Sender) SendPrivate(userID, text string) error {
	s.ctx.SendPrivateMessage(userID, text)
	return nil
}

func (s *Sender) Reply(msg *Message, text string) error {
	if msg.IsPrivate() {
		return s.SendPrivate(msg.UserID, text)
	}
	return s.SendGroup(msg.GroupID, fmt.Sprintf("[CQ:at,qq=%s] %s", msg.UserID, text))
}
```

- [ ] **Step 2: 安装 ZeroBot**

```bash
go get github.com/wdvxdr1123/zerobot
```

- [ ] **Step 3: 编译检查**

```bash
go build ./internal/message/
```
Expected: 编译通过

- [ ] **Step 4: Commit**

```bash
git add internal/message/sender.go go.mod go.sum && git commit -m "feat: message sender with group and private reply"
```

---

### Task 8: Bot 核心——ZeroBot 集成 + 事件处理

**Files:**
- Create: `internal/core/bot.go`

- [ ] **Step 1: 写 bot.go**

```go
// internal/core/bot.go
package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/message"
	"github.com/loveelysia000/robot/internal/session"

	zero "github.com/wdvxdr1123/zerobot"
)

type Bot struct {
	cfg      *Config
	llm      *llm.Client
	session  *session.Manager
	stopChan chan struct{}
}

func NewBot(cfgPath string) (*Bot, error) {
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sessionMgr, err := session.NewManager(db)
	if err != nil {
		return nil, fmt.Errorf("init session: %w", err)
	}

	llmClient := llm.NewClient(llm.DeepSeekConfig{
		APIKey:  cfg.DeepSeek.APIKey,
		BaseURL: cfg.DeepSeek.BaseURL,
		Model:   cfg.DeepSeek.Model,
	})

	return &Bot{
		cfg:      cfg,
		llm:      llmClient,
		session:  sessionMgr,
		stopChan: make(chan struct{}),
	}, nil
}

func (b *Bot) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	engine := zero.New(zero.Config{
		Host:      "0.0.0.0",
		Port:      b.cfg.NapCat.Port,
		AccessToken: b.cfg.NapCat.AccessToken,
	})

	engine.OnRequest(func(c *zero.Ctx) {
		switch c.Event.PostType {
		case "message":
			b.handleMessage(ctx, c)
		case "notice":
			b.handleNotice(ctx, c)
		}
	})

	slog.Info("bot starting", "port", b.cfg.NapCat.Port)
	return engine.Run()
}

func (b *Bot) handleMessage(ctx context.Context, c *zero.Ctx) {
	msg := &message.Message{
		GroupID: fmt.Sprint(c.Event.GroupID),
		UserID:  fmt.Sprint(c.Event.UserID),
		Text:    c.Event.Message,
		IsAtBot: containsAt(c.Event.Message, fmt.Sprint(c.Event.SelfID)),
	}

	cfg := message.TriggerConfig{
		Mode:          b.cfg.Trigger.Mode,
		CommandPrefix: b.cfg.Trigger.CommandPrefix,
	}

	if !message.ShouldReply(msg, cfg) {
		return
	}

	sessionKey := msg.SessionKey()

	// 1. 写入用户消息
	b.session.Append(sessionKey, session.Message{Role: "user", Content: msg.Text})

	// 2. 取窗口历史（含刚刚写入的用户消息）
	history, _ := b.session.GetRecent(sessionKey, 20)

	// 去掉最后一条（即刚刚写入的用户消息，避免重复）
	if len(history) > 0 {
		history = history[:len(history)-1]
	}

	// 3. 转成 llm.ChatMessage
	var chatMsgs []llm.ChatMessage
	for _, h := range history {
		chatMsgs = append(chatMsgs, llm.ChatMessage{Role: h.Role, Content: h.Content})
	}

	// 4. 拼装请求（P1 不带 tools，P3 加上）
	messages := b.llm.BuildMessages(b.cfg.DeepSeek.DefaultSystemPrompt, chatMsgs, msg.Text, nil)

	// 5. 调 DeepSeek
	slog.Info("calling deepseek", "session", sessionKey)
	reply, err := b.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("deepseek error", "err", err)
		reply = "抱歉，我暂时无法回复。"
	}

	// 6. 写入 AI 回复
	b.session.Append(sessionKey, session.Message{Role: "assistant", Content: reply})

	// 7. 发送回复
	sender := message.NewSender(c)
	sender.Reply(msg, reply)
}

func (b *Bot) handleNotice(ctx context.Context, c *zero.Ctx) {
	// P2 实现入群欢迎等事件
}

func containsAt(rawMsg, selfQQ string) bool {
	return strings.Contains(rawMsg, fmt.Sprintf("[CQ:at,qq=%s]", selfQQ))
}
```

- [ ] **Step 2: 编译检查**

```bash
go build ./internal/core/
```
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add internal/core/bot.go && git commit -m "feat: bot core with zerobot integration and message handling pipeline"
```

---

### Task 9: 主入口

**Files:**
- Modify: `cmd/bot/main.go`

- [ ] **Step 1: 写 main.go**

```go
// cmd/bot/main.go
package main

import (
	"log/slog"
	"os"

	"github.com/loveelysia000/robot/internal/core"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	slog.Info("loading config", "path", cfgPath)
	bot, err := core.NewBot(cfgPath)
	if err != nil {
		slog.Error("failed to init bot", "err", err)
		os.Exit(1)
	}

	if err := bot.Run(); err != nil {
		slog.Error("bot stopped", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: 编译**

```bash
go build -o bot ./cmd/bot
```
Expected: 编译通过，生成 `./bot` 二进制

- [ ] **Step 3: 验证二进制可执行**

```bash
./bot --help 2>&1 || true
```
Expected: 读取 config.yaml 或报配置缺失，不panic

- [ ] **Step 4: Commit**

```bash
git add cmd/bot/main.go && git commit -m "feat: main entry point"
```

---

### Task 10: Docker 部署

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`

- [ ] **Step 1: 写 Dockerfile**

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

# 运行阶段
FROM python:3.12-alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bot .
COPY config.yaml .
RUN mkdir -p /app/data

# Python 依赖（P2 角色生成用，P1 先装上）
COPY tools/ ./tools/
COPY prompts/ ./prompts/
RUN pip install --no-cache-dir beautifulsoup4 lxml

EXPOSE 8080
ENTRYPOINT ["./bot"]
```

- [ ] **Step 2: 写 docker-compose.yml**

```yaml
services:
  napcat:
    image: mlikiawa/napcat-docker:latest
    container_name: napcat
    environment:
      - NAPCAT_UID=${NAPCAT_UID:-1000}
      - NAPCAT_GID=${NAPCAT_GID:-1000}
    ports:
      - "6099:6099"
      - "3001:3001"
    volumes:
      - ./napcat/config:/app/napcat/config
      - ./napcat/ntqq:/app/.config/QQ
    restart: always
    network_mode: bridge

  bot:
    build: .
    depends_on: [napcat]
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data
    restart: unless-stopped
```

- [ ] **Step 3: 写 .dockerignore**

```
*.md
.git
.gitignore
docs/
data/
napcat/
```

- [ ] **Step 4: 验证 Dockerfile 构建**

```bash
docker build -t robot-bot:test .
```
Expected: 构建成功

- [ ] **Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml .dockerignore && git commit -m "feat: docker deployment setup"
```

---

### Task 11: 集成编译 + 代码整理

- [ ] **Step 1: 全量编译**

```bash
go build ./...
```
Expected: 所有包编译通过

- [ ] **Step 2: 运行全部测试**

```bash
go test ./... -v
```
Expected: 全部 PASS

- [ ] **Step 3: 运行 go vet**

```bash
go vet ./...
```
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: final integration, all tests pass"
```

---

## 部署验证（手动）

NapCat 侧需在 `./napcat/config/onebot11_{qq}.json` 中配置反向 WS：

```json
{
  "network": {
    "http": { "enable": true, "port": 3000, "accessToken": "change-me" }
  },
  "ws_reverse": [{
    "enable": true,
    "url": "ws://bot:8080",
    "accessToken": "change-me"
  }]
}
```

然后 `docker compose up -d`，通过 `http://<ip>:6099/webui` 扫码登录 QQ。登录后机器人上线，私聊 / @ 机器人发送消息验证。

---

## P1 完成标准

- [ ] `docker compose up -d` 一键启动
- [ ] NapCat 反向 WS 连接稳定
- [ ] 私聊正常对话，带上下文（会话历史不随重启丢失）
- [ ] 群聊 @机器人 正常对话，带上下文
- [ ] SQLite 持久化验证：重启容器后继续對話，历史不丢
- [ ] 所有单元测试通过
