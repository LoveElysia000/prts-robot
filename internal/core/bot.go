// Package core 提供机器人核心功能，包括配置加载、Webhook 服务器、消息处理管道和 QQ API 交互。
package core

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
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
	signKey ed25519.PrivateKey // Ed25519 签名密钥，用于 webhook 回调地址验证
}

// NewBot 从配置文件路径加载配置，初始化所有组件并返回 Bot 实例。
func NewBot(cfgPath string) (*Bot, error) {
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("sqlite", cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// SQLite 不支持并发写入，限制连接数避免 "database is locked" 错误
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

	qqAPI := NewQQAPI(cfg.QQ)

	// 用 app_secret（hex 32字节）作为 Ed25519 种子生成签名密钥
	signKey, err := deriveSignKey(cfg.QQ.AppSecret)
	if err != nil {
		slog.Warn("unable to derive ed25519 sign key, webhook verify will fail", "err", err)
	}

	return &Bot{
		cfg:     cfg,
		qqAPI:   qqAPI,
		llm:     llmClient,
		session: sessionMgr,
		signKey: signKey,
	}, nil
}

// Run 启动 HTTP Webhook 服务器并阻塞，直到进程收到终止信号后优雅关闭。
func (b *Bot) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := b.qqAPI.EnsureToken(); err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

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
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "err", err)
		}
	}()

	slog.Info("bot webhook server starting", "port", b.cfg.QQ.WebhookPort)
	return server.ListenAndServe()
}

// ---- Webhook 数据结构 ----

// WebhookRequest 是 QQ 平台 Webhook 回调的 dispatcher 外层结构。
// 参考 https://bot.q.qq.com/wiki/develop/api-v2/server-inter/event/event-protocol.html
type WebhookRequest struct {
	Op int             `json:"op"` // 0=事件推送, 13=回调地址验证
	T  string          `json:"t"`  // 事件类型，如 "GROUP_AT_MESSAGE_CREATE"
	D  json.RawMessage `json:"d"`  // 事件数据体，根据 op/t 而变化
	ID string          `json:"id"` // 事件 ID
}

// GroupMessageEvent 是群消息事件 d 字段的数据结构。
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

// VerifyRequest 是 op:13 回调地址验证请求的 d 字段数据结构。
type VerifyRequest struct {
	PlainToken string `json:"plain_token"`
	EventTS    string `json:"event_ts"`
}

// VerifyResponse 是 op:13 回调地址验证的响应结构。
type VerifyResponse struct {
	PlainToken string `json:"plain_token"`
	Signature  string `json:"signature"`
}

// ---- Webhook 处理 ----

// handleWebhook 是 QQ 平台 Webhook 回调的统一入口。
// 处理 op:0（事件推送）和 op:13（回调地址验证）两种场景。
func (b *Bot) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("read webhook body failed", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		slog.Error("decode webhook failed", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// op:13 — 回调地址验证
	if req.Op == 13 {
		b.handleVerify(w, body)
		return
	}

	// op:0 — 事件推送：先返回 200 再异步处理
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"code":0}`))

	// 只处理群 @ 消息和群普通消息事件
	if req.T != "GROUP_AT_MESSAGE_CREATE" && req.T != "GROUP_MESSAGE_CREATE" {
		return
	}

	var event GroupMessageEvent
	if err := json.Unmarshal(req.D, &event); err != nil {
		slog.Error("decode event data failed", "err", err)
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

	// 使用独立的 background context，避免 handler 返回后 request context 被取消
	go b.processMessage(context.Background(), msg)
}

// handleVerify 处理 op:13 回调地址验证，使用 Ed25519 签名 plain_token 并返回。
func (b *Bot) handleVerify(w http.ResponseWriter, body []byte) {
	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var verify VerifyRequest
	if err := json.Unmarshal(req.D, &verify); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if b.signKey == nil {
		slog.Error("sign key unavailable, webhook verification will fail")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 签名内容 = event_ts + plain_token
	signData := verify.EventTS + verify.PlainToken
	signature := ed25519.Sign(b.signKey, []byte(signData))

	resp := VerifyResponse{
		PlainToken: verify.PlainToken,
		Signature:  hex.EncodeToString(signature),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ---- 消息处理 ----

// processMessage 处理收到的消息：写会话 → 取历史窗口 → 调 LLM → 发回复 → 存回复。
func (b *Bot) processMessage(ctx context.Context, msg *message.Message) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionKey := msg.SessionKey()

	if err := b.session.Append(sessionKey, session.Message{Role: "user", Content: msg.Text}); err != nil {
		slog.Warn("append user message failed", "err", err)
	}

	history, err := b.session.GetRecent(sessionKey, 20)
	if err != nil {
		slog.Warn("get recent messages failed, continuing without history", "err", err)
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

	// 先发送回复，成功后再存入会话，避免发送失败导致上下文错位
	sendErr := b.qqAPI.SendGroupMessage(msg.GroupID, reply, msg.MsgID)
	if sendErr != nil {
		slog.Error("send reply failed", "err", sendErr)
	}

	if err := b.session.Append(sessionKey, session.Message{Role: "assistant", Content: reply}); err != nil {
		slog.Warn("append assistant message failed", "err", err)
	}
}

// ---- 签名工具 ----

// deriveSignKey 从 QQ 机器人 app_secret（hex 格式 32 字节）生成 Ed25519 签名密钥。
func deriveSignKey(appSecret string) (ed25519.PrivateKey, error) {
	seed, err := hex.DecodeString(appSecret)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("app_secret must be hex 32 bytes, got %d bytes: %w", len(seed), err)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}
