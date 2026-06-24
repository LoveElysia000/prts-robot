// Package core 提供机器人核心功能，包括配置加载、ZeroBot 驱动和消息处理。
package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	_ "modernc.org/sqlite"

	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/driver"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/session"
)

var reCQCode = regexp.MustCompile(`\[CQ:[^]]+]`)

// Bot 是机器人主控结构体。
type Bot struct {
	cfg     *Config
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

	return &Bot{
		cfg:     cfg,
		llm:     llmClient,
		session: sessionMgr,
	}, nil
}

// Run 启动 ZeroBot，通过反向 WebSocket 接收 NapCat 推送的消息。
func (b *Bot) Run() {
	zero.OnMessage().SetBlock(false).Handle(b.handleMessage)

	slog.Info("bot starting (NapCat/ZeroBot reverse WS)")
	zero.RunAndBlock(&zero.Config{
		SuperUsers: []int64{},
		Driver: []zero.Driver{
			driver.NewWebSocketServer(16, "ws://0.0.0.0:8080", b.cfg.NapCat.AccessToken),
		},
	}, nil)
}

// handleMessage 处理收到的每一条消息。
func (b *Bot) handleMessage(ctx *zero.Ctx) {
	text := ctx.Event.Message.String()
	if text == "" {
		return
	}

	isPrivate := ctx.Event.MessageType == "private"
	isGroup := ctx.Event.MessageType == "group"
	if !isPrivate && !isGroup {
		return
	}

	isAtBot := isPrivate || isMentioned(ctx)
	text = reCQCode.ReplaceAllString(text, "")
	if !isAtBot && b.cfg.Trigger.Mode != "all" {
		return
	}

	go b.processMessage(context.Background(), text, isPrivate, ctx)
}

// isMentioned 判断群消息中是否 @了机器人。
func isMentioned(ctx *zero.Ctx) bool {
	selfID := strconv.FormatInt(ctx.Event.SelfID, 10)
	for _, seg := range ctx.Event.Message {
		if seg.Type != "at" {
			continue
		}
		qq, ok := seg.Data["qq"]
		if !ok {
			continue
		}
		if fmt.Sprint(qq) == "all" || fmt.Sprint(qq) == selfID {
			return true
		}
	}
	return false
}

// processMessage 处理消息。
func (b *Bot) processMessage(ctx context.Context, text string, isPrivate bool, c *zero.Ctx) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionKey := fmt.Sprintf("group_%d", c.Event.GroupID)
	if isPrivate {
		sessionKey = fmt.Sprintf("private_%d", c.Event.UserID)
	}

	b.session.Append(sessionKey, session.Message{Role: "user", Content: text})

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

	messages := b.llm.BuildMessages(b.cfg.DeepSeek.DefaultSystemPrompt, chatMsgs, text, nil)

	slog.Info("calling deepseek", "session", sessionKey)
	reply, err := b.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("deepseek error", "err", err)
		reply = "抱歉，我暂时无法回复。"
	}

	if isPrivate {
		c.SendPrivateMessage(c.Event.UserID, reply)
	} else {
		c.SendGroupMessage(c.Event.GroupID, reply)
	}

	b.session.Append(sessionKey, session.Message{Role: "assistant", Content: reply})
}
