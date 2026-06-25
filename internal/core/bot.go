package core

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/persona"
	"github.com/loveelysia000/robot/internal/session"
)

type Bot struct {
	cfg     *Config
	llm     *llm.Client
	session *session.Manager
	dg      *discordgo.Session
	persona *persona.Manager
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

	personaMgr, err := persona.NewManager("data/personas.yaml", cfg.DeepSeek.DefaultSystemPrompt)
	if err != nil {
		slog.Warn("persona manager init failed, using default prompt only", "err", err)
		personaMgr = nil
	}

	return &Bot{
		cfg:     cfg,
		llm:     llmClient,
		session: sessionMgr,
		persona: personaMgr,
	}, nil
}

func (b *Bot) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dg, err := discordgo.New("Bot " + b.cfg.Discord.BotToken)
	if err != nil {
		return fmt.Errorf("discord session: %w", err)
	}
	b.dg = dg

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent
	dg.AddHandler(b.handleMessage)

	if err := dg.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	defer dg.Close()

	slog.Info("bot started (Discord)", "user", dg.State.User.Username)
	<-ctx.Done()
	slog.Info("bot shutting down")
	return nil
}

func (b *Bot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	isDM := m.GuildID == ""
	isMentioned := false
	if !isDM {
		for _, u := range m.Mentions {
			if u.ID == s.State.User.ID {
				isMentioned = true
				break
			}
		}
	}

	slog.Debug("received message",
		"guildID", m.GuildID,
		"channelID", m.ChannelID,
		"author", m.Author.Username,
		"isDM", isDM,
		"isMentioned", isMentioned,
		"content", m.Content,
	)

	if !isDM && !isMentioned && b.cfg.Trigger.Mode != "all" {
		return
	}

	text := strings.TrimSpace(m.ContentWithMentionsReplaced())
	slog.Debug("cleaned text", "text", text)

	go b.processMessage(context.Background(), text, isDM, m, s)
}

func (b *Bot) processMessage(ctx context.Context, text string, isDM bool, m *discordgo.MessageCreate, s *discordgo.Session) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sessionKey := m.GuildID
	if isDM {
		sessionKey = "dm_" + m.Author.ID
	}

	b.session.Append(sessionKey, session.Message{Role: "user", Content: text})

	history, err := b.session.GetRecent(sessionKey, 20)
	if err != nil {
		slog.Warn("get recent failed", "err", err)
	}
	if len(history) > 0 {
		history = history[:len(history)-1]
	}

	var chatMsgs []llm.ChatMessage
	for _, h := range history {
		chatMsgs = append(chatMsgs, llm.ChatMessage{Role: h.Role, Content: h.Content})
	}

	systemPrompt := b.cfg.DeepSeek.DefaultSystemPrompt
	if b.persona != nil {
		systemPrompt = b.persona.GetForChannel(m.ChannelID)
	}
	messages := b.llm.BuildMessages(systemPrompt, chatMsgs, text, nil)

	slog.Info("calling deepseek", "session", sessionKey)
	reply, err := b.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("deepseek error", "err", err)
		reply = "抱歉，我暂时无法回复。"
	}

	s.ChannelMessageSendReply(m.ChannelID, reply, m.Reference())
	b.session.Append(sessionKey, session.Message{Role: "assistant", Content: reply})
}
