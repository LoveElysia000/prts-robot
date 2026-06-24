// Package core 提供机器人核心功能，包括配置加载、Webhook 服务器、消息处理管道和 QQ API 交互。
package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "modernc.org/sqlite"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/message"
	"github.com/loveelysia000/robot/internal/session"
)

// Bot 是机器人主控结构体，管理配置、QQ API、LLM 客户端和会话管理器。
type Bot struct {
	cfg     *Config
	qqAPI   *QQAPI
	llm     *llm.Client
	session *session.Manager
}

// NewBot 从配置文件加载并初始化 Bot。
func NewBot(cfgPath string) (*Bot, error) {
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	sessionMgr, err := session.NewManager(db)
	if err != nil {
		return nil, fmt.Errorf("init session: %w", err)
	}

	llmClient := llm.NewClient(llm.DeepSeekConfig{
		APIKey:  cfg.DeepSeek.APIKey,
		BaseURL: cfg.DeepSeek.BaseURL,
		Model:   cfg.DeepSeek.Model,
	})

	qqAPI := NewQQAPI(cfg.QQ.AppID, cfg.QQ.AppSecret)
	if cfg.QQ.AppSecret == "" {
		return nil, fmt.Errorf("QQ_APP_SECRET environment variable is not set")
	}

	return &Bot{
		cfg:     cfg,
		qqAPI:   qqAPI,
		llm:     llmClient,
		session: sessionMgr,
	}, nil
}

// Run 启动 HTTP Webhook 服务器并阻塞，直到收到终止信号。
func (b *Bot) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", b.handleWebhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", b.cfg.QQ.WebhookPort),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	slog.Info("bot webhook server starting", "port", b.cfg.QQ.WebhookPort)
	return server.ListenAndServe()
}

// ---- Webhook 数据结构 ----

// WebhookRequest 是 QQ 平台 Webhook 的 dispatcher 外层结构。
type WebhookRequest struct {
	Op int             `json:"op"` // 0=事件推送, 13=回调验证
	T  string          `json:"t"`  // 事件类型
	D  json.RawMessage `json:"d"`  // 事件数据
	ID string          `json:"id"`
}

// GroupMessageEvent 是群消息事件的数据结构。
type GroupMessageEvent struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	GroupID   string         `json:"group_openid"`
	Author    GroupMsgAuthor `json:"author"`
	Timestamp string         `json:"timestamp"`
}

// GroupMsgAuthor 是群消息作者信息。
type GroupMsgAuthor struct {
	ID string `json:"member_openid"`
}

// ---- Webhook 处理 ----

// handleWebhook 处理 QQ 平台的 Webhook 回调，op:0 事件推送，op:13 回调验证。
func (b *Bot) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("read webhook failed", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		slog.Error("decode webhook failed", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// op:13 — 回调地址验证，直接回显 plain_token
	if req.Op == 13 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body) // 直接原样返回，QQ 平台会校验
		return
	}

	// op:0 — 事件推送，先返回 200
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"code":0}`))

	if req.T != "GROUP_AT_MESSAGE_CREATE" && req.T != "GROUP_MESSAGE_CREATE" {
		return
	}

	var event GroupMessageEvent
	if err := json.Unmarshal(req.D, &event); err != nil {
		slog.Error("decode event failed", "err", err)
		return
	}

	msg := &message.Message{
		GroupID: event.GroupID,
		UserID:  event.Author.ID,
		Text:    event.Content,
		MsgID:   event.ID,
		IsAtBot: req.T == "GROUP_AT_MESSAGE_CREATE",
	}

	cfg := message.TriggerConfig{
		Mode:          b.cfg.Trigger.Mode,
		CommandPrefix: b.cfg.Trigger.CommandPrefix,
	}

	if !message.ShouldReply(msg, cfg) {
		return
	}

	go b.processMessage(context.Background(), msg)
}

// ---- 消息处理 ----

// processMessage 处理消息：写会话 → 取窗口 → LLM → 发回复 → 存回复。
func (b *Bot) processMessage(ctx context.Context, msg *message.Message) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionKey := msg.SessionKey()

	if err := b.session.Append(sessionKey, session.Message{Role: "user", Content: msg.Text}); err != nil {
		slog.Warn("append user message failed", "err", err)
	}

	history, err := b.session.GetRecent(sessionKey, 20)
	if err != nil {
		slog.Warn("get recent messages failed", "err", err)
	}
	if len(history) > 0 {
		history = history[:len(history)-1]
	}

	var chatMsgs []llm.ChatMessage
	for _, h := range history {
		chatMsgs = append(chatMsgs, llm.ChatMessage{Role: h.Role, Content: h.Content})
	}

	messages := b.llm.BuildMessages(b.cfg.DeepSeek.DefaultSystemPrompt, chatMsgs, msg.Text, nil)

	slog.Info("calling deepseek", "session", sessionKey)
	reply, err := b.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("deepseek error", "err", err)
		reply = "抱歉，我暂时无法回复。"
	}

	if err := b.qqAPI.SendGroupMessage(msg.GroupID, reply, msg.MsgID); err != nil {
		slog.Error("send reply failed", "err", err)
	}

	if err := b.session.Append(sessionKey, session.Message{Role: "assistant", Content: reply}); err != nil {
		slog.Warn("append assistant message failed", "err", err)
	}
}
