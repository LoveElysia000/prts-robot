// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
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

// Bot 是机器人主控结构体，管理配置、QQ API、LLM 客户端和会话管理器。
type Bot struct {
	cfg     *Config
	qqAPI   *QQAPI
	llm     *llm.Client
	session *session.Manager
}

// NewBot 从配置路径加载配置，初始化数据库、会话管理器、LLM 客户端和 QQ API，返回 Bot 实例。
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

// Run 启动机器人 webhook HTTP 服务器，监听配置端口并处理传入消息。
func (b *Bot) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := b.qqAPI.EnsureToken(); err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", b.handleWebhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
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

// WebhookPayload 表示 QQ 机器人 webhook 回调的 JSON 负载结构。
type WebhookPayload struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	Content   string `json:"content"`
	GroupID   string `json:"group_openid"`
	Author    struct {
		ID string `json:"member_openid"`
	} `json:"author"`
	Timestamp string `json:"timestamp"`
}

// handleWebhook 处理来自 QQ 平台的 webhook 回调请求，解析消息内容并决定是否需要回复。
func (b *Bot) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"code":0}`))

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("decode webhook failed", "err", err)
		return
	}

	if payload.Type != 0 {
		return
	}

	msg := &message.Message{
		GroupID: payload.GroupID,
		UserID:  payload.Author.ID,
		Text:    payload.Content,
		MsgID:   payload.ID,
		IsAtBot: strings.Contains(payload.Content, "@机器人"),
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

// processMessage 处理单条消息，将会话历史与当前消息拼接后调用 LLM 生成回复，并将回复发送到 QQ 群。
func (b *Bot) processMessage(ctx context.Context, msg *message.Message) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionKey := msg.SessionKey()

	b.session.Append(sessionKey, session.Message{Role: "user", Content: msg.Text})

	history, _ := b.session.GetRecent(sessionKey, 20)
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

	b.session.Append(sessionKey, session.Message{Role: "assistant", Content: reply})

	if err := b.qqAPI.SendGroupMessage(msg.GroupID, reply, msg.MsgID); err != nil {
		slog.Error("send reply failed", "err", err)
	}
}
