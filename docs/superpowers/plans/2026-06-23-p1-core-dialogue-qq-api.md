# P1: 核心对话 实现计划（官方 QQ API）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 机器人上线，通过官方 QQ 机器人 API 接收群消息，带上下文的 AI 对话，SQLite 持久化会话历史，Docker 一键启动。

**Architecture:** `net/http` 起 Webhook 服务器接收 QQ 官方推送，消息经触发判断后走 AI 管线（SQLite 取最近 20 轮历史 + system prompt → DeepSeek API），通过官方发消息 API 回复。

**Tech Stack:** Go 1.22+, net/http（标准库）, go-openai, modernc.org/sqlite, yaml.v3

**与 NapCat 版计划的差异：** 仅消息收发层不同。session、llm、message types、config loading 等模块完全一致。

---

## 文件结构

```
robot/
├── cmd/bot/main.go                  # 入口
├── internal/
│   ├── core/
│   │   ├── config.go                # 配置加载
│   │   ├── bot.go                   # HTTP Webhook 服务器 + 事件处理
│   │   └── qqapi.go                 # QQ 官方 API 客户端（发消息/获取 token）
│   ├── message/
│   │   ├── types.go                 # 消息结构体
│   │   └── handler.go               # 触发判断 + 消息处理管线
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
qq:
  app_id: "your-app-id"
  app_secret: "your-app-secret"
  webhook_port: 8080

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
git add -A && git commit -m "chore: project scaffolding for official QQ API"
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
qq:
  app_id: "123"
  app_secret: "secret123"
  webhook_port: 8080
deepseek:
  api_key: "sk-test"
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  default_system_prompt: "test"
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
	if cfg.QQ.AppID != "123" {
		t.Errorf("expected app_id 123, got %s", cfg.QQ.AppID)
	}
	if cfg.DeepSeek.Model != "deepseek-v4-flash" {
		t.Errorf("expected model deepseek-v4-flash, got %s", cfg.DeepSeek.Model)
	}
	if cfg.Trigger.Mode != "hybrid" {
		t.Errorf("expected mode hybrid, got %s", cfg.Trigger.Mode)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/core/ -v -run TestLoadConfig
```
Expected: FAIL

- [ ] **Step 3: 写 config.go**

```go
// internal/core/config.go
package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	QQ       QQConfig       `yaml:"qq"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

type QQConfig struct {
	AppID       string `yaml:"app_id"`
	AppSecret   string `yaml:"app_secret"`
	WebhookPort int    `yaml:"webhook_port"`
}

type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`
	BaseURL             string `yaml:"base_url"`
	Model               string `yaml:"model"`
	DefaultSystemPrompt string `yaml:"default_system_prompt"`
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
git add internal/core/ go.mod go.sum && git commit -m "feat: config loading for official QQ API"
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
	// 官方 API 只有群聊，没有私聊
	msgGroup := &Message{GroupID: "12345"}
	if key := msgGroup.SessionKey(); key != "group_12345" {
		t.Errorf("expected group_12345, got %s", key)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/message/ -v
```
Expected: FAIL

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
	MsgID   string // 官方 API 消息 ID，用于回复时引用
}

func (m *Message) IsCommand(prefix string) bool {
	return strings.HasPrefix(m.Text, prefix)
}

func (m *Message) SessionKey() string {
	return "group_" + m.GroupID
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/message/ -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/message/ && git commit -m "feat: message types for official QQ API"
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
}

func TestGetRecentEmpty(t *testing.T) {
	db := setupDB(t)
	mgr, _ := NewManager(db)
	recent, _ := mgr.GetRecent("group_nonexist", 10)
	if len(recent) != 0 {
		t.Errorf("expected 0 messages, got %d", len(recent))
	}
}
```

- [ ] **Step 3: 运行测试，确认失败**

```bash
go test ./internal/session/ -v
```
Expected: FAIL

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
git add internal/session/ go.mod go.sum && git commit -m "feat: SQLite session manager"
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
	api   *openai.Client
	model string
}

func NewClient(cfg DeepSeekConfig) *Client {
	api := openai.NewClientWithConfig(openai.DefaultConfig(cfg.APIKey))
	api.BaseURL = cfg.BaseURL
	return &Client{api: api, model: cfg.Model}
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
		return "", fmt.Errorf("empty response")
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

### Task 6: 触发判断

**Files:**
- Create: `internal/message/handler.go`
- Create: `internal/message/handler_test.go`

- [ ] **Step 1: 写 handler_test.go**

```go
// internal/message/handler_test.go
package message

import "testing"

type TriggerConfig struct {
	Mode          string
	CommandPrefix string
}

func TestShouldReply(t *testing.T) {
	cfg := TriggerConfig{Mode: "hybrid", CommandPrefix: "/"}

	// hybrid: 群@才回
	if !ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: true}, cfg) {
		t.Error("hybrid: at msg should reply")
	}
	if ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfg) {
		t.Error("hybrid: non-at should not reply")
	}

	// all
	cfgAll := TriggerConfig{Mode: "all", CommandPrefix: "/"}
	if !ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfgAll) {
		t.Error("all: should reply")
	}

	// at
	cfgAt := TriggerConfig{Mode: "at", CommandPrefix: "/"}
	if ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfgAt) {
		t.Error("at: non-at should not reply")
	}
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
git add internal/message/handler.go internal/message/handler_test.go && git commit -m "feat: trigger logic"
```

---

### Task 7: QQ 官方 API 客户端

**Files:**
- Create: `internal/core/qqapi.go`
- Create: `internal/core/qqapi_test.go`

- [ ] **Step 1: 写 qqapi_test.go**

```go
// internal/core/qqapi_test.go
package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendGroupMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Error("expected POST")
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing auth header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123"}`))
	}))
	defer server.Close()

	api := &QQAPI{
		baseURL:   server.URL,
		accessToken: "test-token",
	}
	err := api.SendGroupMessage("group_123", "你好")
	if err != nil {
		t.Fatalf("SendGroupMessage failed: %v", err)
	}
}

func TestGetAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"token_abc","expires_in":7200}`))
	}))
	defer server.Close()

	api := &QQAPI{
		baseURL:    server.URL,
		appID:      "app123",
		appSecret:  "secret123",
	}
	token, err := api.getAccessToken()
	if err != nil {
		t.Fatalf("getAccessToken failed: %v", err)
	}
	if token != "token_abc" {
		t.Errorf("expected token_abc, got %s", token)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./internal/core/ -v -run "TestSendGroupMessage|TestGetAccessToken"
```
Expected: FAIL

- [ ] **Step 3: 写 qqapi.go**

```go
// internal/core/qqapi.go
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type QQAPI struct {
	baseURL     string
	appID       string
	appSecret   string
	accessToken string
	mu          sync.RWMutex
}

func NewQQAPI(cfg QQConfig) *QQAPI {
	return &QQAPI{
		baseURL:   "https://api.sgroup.qq.com",
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
	}
}

func (q *QQAPI) EnsureToken() error {
	q.mu.RLock()
	if q.accessToken != "" {
		q.mu.RUnlock()
		return nil
	}
	q.mu.RUnlock()

	q.mu.Lock()
	defer q.mu.Unlock()

	token, err := q.getAccessToken()
	if err != nil {
		return err
	}
	q.accessToken = token

	// 定时刷新（提前 10 分钟）
	go func() {
		time.Sleep(1*time.Hour + 50*time.Minute)
		q.mu.Lock()
		q.accessToken = ""
		q.mu.Unlock()
	}()

	return nil
}

func (q *QQAPI) getAccessToken() (string, error) {
	resp, err := http.Post(
		fmt.Sprintf("https://bots.qq.com/app/getAppAccessToken?appId=%s&clientSecret=%s", q.appID, q.appSecret),
		"application/json",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access_token")
	}
	return result.AccessToken, nil
}

func (q *QQAPI) SendGroupMessage(groupID, content, replyMsgID string) error {
	q.mu.RLock()
	token := q.accessToken
	q.mu.RUnlock()

	body := map[string]any{
		"content": content,
		"msg_type": 0,
		"msg_id": replyMsgID, // 回复消息时引用原消息
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/v2/groups/%s/messages", q.baseURL, groupID),
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("QQBot %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send message failed: status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/core/ -v -run "TestSendGroupMessage|TestGetAccessToken"
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/qqapi.go internal/core/qqapi_test.go && git commit -m "feat: QQ official API client"
```

---

### Task 8: Bot 核心——Webhook 服务器 + 消息处理

**Files:**
- Create: `internal/core/bot.go`

- [ ] **Step 1: 写 bot.go**

```go
// internal/core/bot.go
package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/message"
	"github.com/loveelysia000/robot/internal/session"
)

type Bot struct {
	cfg     *Config
	qqAPI   *QQAPI
	llm     *llm.Client
	session *session.Manager
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

	qqAPI := NewQQAPI(cfg.QQ)

	return &Bot{
		cfg:     cfg,
		qqAPI:   qqAPI,
		llm:     llmClient,
		session: sessionMgr,
	}, nil
}

func (b *Bot) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// 初始化 access token（含定时刷新）
	if err := b.qqAPI.EnsureToken(); err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", b.handleWebhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", b.cfg.QQ.WebhookPort),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	slog.Info("bot webhook server starting", "port", b.cfg.QQ.WebhookPort)
	return server.ListenAndServe()
}

// QQ Webhook 推送的消息结构（官方 API 格式）
type WebhookPayload struct {
	ID        string `json:"id"`
	Type      int    `json:"type"` // 0=普通消息, 更多 type 见官方文档
	Content   string `json:"content"`
	GroupID   string `json:"group_openid"`
	Author    struct {
		ID string `json:"member_openid"`
	} `json:"author"`
	Timestamp string `json:"timestamp"`
}

func (b *Bot) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// 官方 API 需要在 5 秒内返回 200，否则会重试
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"code":0}`))

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("decode webhook failed", "err", err)
		return
	}

	// 只处理普通消息
	if payload.Type != 0 {
		return
	}

	msg := &message.Message{
		GroupID: payload.GroupID,
		UserID:  payload.Author.ID,
		Text:    payload.Content,
		MsgID:   payload.ID,
		IsAtBot: strings.Contains(payload.Content, "@机器人"), // 简化判断，P2 改为解析 mentions 字段
	}

	cfg := message.TriggerConfig{
		Mode:          b.cfg.Trigger.Mode,
		CommandPrefix: b.cfg.Trigger.CommandPrefix,
	}

	if !message.ShouldReply(msg, cfg) {
		return
	}

	go b.processMessage(r.Context(), msg)
}

func (b *Bot) processMessage(ctx context.Context, msg *message.Message) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionKey := msg.SessionKey()

	// 1. 写入用户消息
	b.session.Append(sessionKey, session.Message{Role: "user", Content: msg.Text})

	// 2. 取窗口历史
	history, _ := b.session.GetRecent(sessionKey, 20)
	if len(history) > 0 {
		history = history[:len(history)-1] // 去掉刚写入的当前消息
	}

	// 3. 转成 llm.ChatMessage
	var chatMsgs []llm.ChatMessage
	for _, h := range history {
		chatMsgs = append(chatMsgs, llm.ChatMessage{Role: h.Role, Content: h.Content})
	}

	// 4. 拼装请求
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
	if err := b.qqAPI.SendGroupMessage(msg.GroupID, reply, msg.MsgID); err != nil {
		slog.Error("send reply failed", "err", err)
	}
}
```

- [ ] **Step 2: 编译检查**

```bash
go build ./internal/core/
```
Expected: 编译通过

- [ ] **Step 3: Commit**

```bash
git add internal/core/bot.go && git commit -m "feat: bot core with webhook server and message processing"
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

- [ ] **Step 3: Commit**

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
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

FROM python:3.12-alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bot .
COPY config.yaml .
RUN mkdir -p /app/data

COPY tools/ ./tools/
COPY prompts/ ./prompts/
RUN pip install --no-cache-dir beautifulsoup4 lxml

EXPOSE 8080
ENTRYPOINT ["./bot"]
```

- [ ] **Step 2: 写 docker-compose.yml**

```yaml
services:
  bot:
    build: .
    ports:
      - "8080:8080"
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
```

- [ ] **Step 4: 验证构建**

```bash
docker build -t robot-bot:test .
```
Expected: 构建成功

- [ ] **Step 5: Commit**

```bash
git add Dockerfile docker-compose.yml .dockerignore && git commit -m "feat: docker deployment for official QQ API"
```

---

### Task 11: 集成编译 + 代码整理

- [ ] **Step 1: 全量编译**

```bash
go build ./...
```
Expected: 全部编译通过

- [ ] **Step 2: 运行全部测试**

```bash
go test ./... -v
```
Expected: PASS

- [ ] **Step 3: 运行 go vet**

```bash
go vet ./...
```
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: integration compile, all tests pass"
```

---

## 部署验证（手动）

1. 在 QQ 开放平台 (https://q.qq.com) 创建机器人，获取 AppID、AppSecret
2. 配置 Webhook 地址为 `http://<你的服务器IP>:8080/webhook`
3. 填写 `config.yaml` 中的 qq 字段
4. `docker compose up -d`
5. 在群里 @机器人 发送消息验证

---

## P1 完成标准

- [ ] `docker compose up -d` 一键启动
- [ ] Webhook 接收 QQ 群消息正常
- [ ] 群聊 @机器人 正常对话，带上下文
- [ ] SQLite 持久化验证：重启容器后对话历史不丢
- [ ] 所有单元测试通过
