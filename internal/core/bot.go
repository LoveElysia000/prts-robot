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
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/persona"
	"github.com/loveelysia000/robot/internal/persona/generator"
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

	// 提取命令文本（去掉 @mention 前缀）
	cmdText := text
	if isMentioned && !isDM {
		// 去掉消息中的 @botname 部分，保留后面的命令
		if idx := strings.Index(text, " "); idx > 0 {
			cmdText = strings.TrimSpace(text[idx+1:])
		}
	}

	// 命令路由
	if strings.HasPrefix(cmdText, "/") {
		go b.handleCommand(s, m, cmdText)
		return
	}

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

func (b *Bot) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate, cmdText string) {
	parts := strings.Fields(cmdText)
	if len(parts) == 0 {
		return
	}

	// 归一化：/角色列表 → /角色 列表 等
	cmd, subArgs := normalizeCommand(parts[0])
	args := append(subArgs, parts[1:]...)

	reply := b.runCommand(cmd, args, m.ChannelID)
	if reply != "" {
		s.ChannelMessageSendReply(m.ChannelID, reply, m.Reference())
	}
}

// normalizeCommand 把 "/角色列表" "/角色切换" 等合并命令拆成 "/角色" + ["列表"]
func normalizeCommand(raw string) (cmd string, args []string) {
	if strings.HasPrefix(raw, "/角色列表") {
		return "/角色", []string{"列表"}
	}
	if strings.HasPrefix(raw, "/角色切换") {
		return "/角色", []string{"切换"}
	}
	if strings.HasPrefix(raw, "/角色重载") {
		return "/角色", []string{"重载"}
	}
	return raw, nil
}

func (b *Bot) runCommand(cmd string, args []string, channelID string) string {
	switch cmd {
	case "/角色":
		return b.cmdRole(args, channelID)
	case "/角色校正":
		return b.cmdCorrect(args)
	case "/help":
		return "**可用命令:**\n/角色 列表 — 查看角色\n/角色 切换 <名字> — 切换角色\n/角色 重载 — 热加载配置\n/角色校正 <名字> <内容> — 修正人设\n/生成角色 <名字> <URL> — 生成角色"
	case "/生成角色":
		return b.cmdGenerate(args, channelID)
	default:
		return "未知命令，输入 /help 查看"
	}
}

func (b *Bot) cmdRole(args []string, channelID string) string {
	if b.persona == nil {
		return "角色系统未启用"
	}
	// normalizeCommand 已经把 /角色列表 /角色切换 拆成了 args[0]="列表"/"切换"/"重载"
	if len(args) == 0 {
		return "用法: /角色 列表 | 切换 <slug> | 重载"
	}
	switch args[0] {
	case "列表":
		list := b.persona.List()
		if len(list) == 0 {
			return "暂无可用角色"
		}
		return "已注册角色:\n" + strings.Join(list, "\n")
	case "切换":
		if len(args) < 2 {
			return "用法: /角色 切换 <角色名或slug>"
		}
		p, ok := b.persona.FindPersona(args[1])
		if !ok {
			return fmt.Sprintf("角色 %s 不存在，输入 /角色 列表 查看", args[1])
		}
		b.updateBinding(channelID, p.Slug)
		return fmt.Sprintf("已切换到 %s", p.Name)
	case "重载":
		if err := b.persona.Reload(); err != nil {
			return fmt.Sprintf("重载失败: %v", err)
		}
		return "角色配置已重载"
	default:
		return "用法: /角色 列表 | 切换 <slug> | 重载"
	}
}

func (b *Bot) cmdCorrect(args []string) string {
	if b.persona == nil {
		return "角色系统未启用"
	}
	if len(args) < 2 {
		return "用法: /角色校正 <角色名或slug> <修正指令>"
	}
	p, ok := b.persona.FindPersona(args[0])
	if !ok {
		return fmt.Sprintf("角色 %s 不存在", args[0])
	}
	instruction := strings.Join(args[1:], " ")
	if err := b.persona.Correct(context.Background(), b.llm, p.Slug, instruction); err != nil {
		return fmt.Sprintf("校正失败: %v", err)
	}
	b.persona.Reload()
	return fmt.Sprintf("角色 %s 已校正", p.Name)
}

func (b *Bot) cmdGenerate(args []string, _ string) string {
	if len(args) < 2 {
		return "用法: /生成角色 <slug> <Wiki URL>"
	}
	slug, url := args[0], args[1]
	go func() {
		gen := generator.NewGenerator(b.llm)
		if err := gen.Generate(context.Background(), generator.GenerateRequest{
			Slug: slug, Name: slug, WikiURL: url,
		}); err != nil {
			slog.Error("generate failed", "slug", slug, "err", err)
			return
		}
		b.registerPersona(slug, slug)
		b.persona.Reload()
	}()
	return fmt.Sprintf("正在生成角色 %s ...（约 20 秒）", slug)
}

// registerPersona 将角色注册到 personas.yaml（如不存在）。
func (b *Bot) registerPersona(slug, name string) {
	data, err := os.ReadFile("data/personas.yaml")
	if err != nil {
		slog.Error("registerPersona: read failed", "err", err)
		return
	}
	var cfg persona.PersonaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		slog.Error("registerPersona: parse failed", "err", err)
		return
	}
	if cfg.Personas == nil {
		cfg.Personas = make(map[string]struct {
			Name     string   `yaml:"name"`
			SkillDir string   `yaml:"skill_dir"`
			Skills   []string `yaml:"skills"`
		})
	}
	if _, exists := cfg.Personas[slug]; exists {
		return
	}
	cfg.Personas[slug] = struct {
		Name     string   `yaml:"name"`
		SkillDir string   `yaml:"skill_dir"`
		Skills   []string `yaml:"skills"`
	}{
		Name:     name,
		SkillDir: "data/personas/" + slug,
		Skills:   []string{},
	}
	out, _ := yaml.Marshal(&cfg)
	if err := os.WriteFile("data/personas.yaml", out, 0644); err != nil {
		slog.Error("registerPersona: write failed", "err", err)
	}
}

func (b *Bot) updateBinding(channelID, slug string) {
	data, err := os.ReadFile("data/personas.yaml")
	if err != nil {
		slog.Error("read personas.yaml failed", "err", err)
		return
	}
	var cfg persona.PersonaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		slog.Error("parse personas.yaml failed", "err", err)
		return
	}
	if cfg.Bindings == nil {
		cfg.Bindings = make(map[string]string)
	}
	cfg.Bindings[channelID] = slug
	out, _ := yaml.Marshal(&cfg)
	os.WriteFile("data/personas.yaml", out, 0644)
	b.persona.Reload()
}
