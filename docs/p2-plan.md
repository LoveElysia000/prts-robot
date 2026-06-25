# P2: 角色系统 + 命令路由 实现计划

> **Goal:** 加载 SKILL.md 替代默认 prompt，实现按频道绑定角色、命令系统、角色生成。

**Architecture:** PersonaManager 从 `data/personas/` 加载角色文件，按频道绑定；消息以 `/` 开头走命令路由，否则走 AI 对话（注入角色 system prompt）。

**Tech Stack:** Go 1.25, discordgo, DeepSeek API, Python 3.12 (子进程)

---

## 文件变更

```
新增:
  internal/persona/
    ├── manager.go       # 角色加载、绑定、热加载
    ├── card.go          # 角色卡结构定义
    ├── corrector.go     # AI 局部校正 (/角色校正)
    └── generator/
        ├── generator.go     # /生成角色 流程编排
        ├── fetcher.go       # Go 抓取 Wiki 页面
        ├── parser_bridge.go # exec 调用 Python prts_parser.py
        └── writer.go        # exec 调用 Python character_skill_writer.py

修改:
  internal/core/bot.go       # 集成 PersonaManager + 命令路由
  internal/core/config.go    # 添加 persona 配置项
  config.example.yaml        # 加 persona 段

复用（已有）:
  internal/session/manager.go
  internal/llm/client.go
  internal/llm/client.go (BuildMessages 已支持 system prompt)
  tools/prts_parser.py
  tools/character_skill_writer.py
  prompts/*.md
  data/personas/lin/SKILL.md (347行完整版)
```

---

### Task 1: 角色卡定义

**Files:**
- Create: `internal/persona/card.go`

```go
// internal/persona/card.go
package persona

// Persona 表示一个角色。
type Persona struct {
	Name     string   // 角色名
	Slug     string   // 标识，如 "lin"
	SkillDir string   // SKILL.md 所在目录
	Prompt   string   // 加载后的 SKILL.md 内容
	Skills   []string // 拥有的 Function 工具（P3 用）
}

// LoadPrompt 读取 SKILL.md 文件内容。
func (p *Persona) LoadPrompt() error {
	data, err := os.ReadFile(filepath.Join(p.SkillDir, "SKILL.md"))
	if err != nil {
		return err
	}
	p.Prompt = string(data)
	return nil
}
```

- [ ] Compile: `go build ./internal/persona/`

---

### Task 2: PersonaManager

**Files:**
- Create: `internal/persona/manager.go`

```go
// internal/persona/manager.go
package persona

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type PersonaConfig struct {
	Personas map[string]struct {
		Name     string   `yaml:"name"`
		SkillDir string   `yaml:"skill_dir"`
		Skills   []string `yaml:"skills"`
	} `yaml:"personas"`
	Bindings map[string]string `yaml:"bindings"` // channelID -> slug
}

type Manager struct {
	mu          sync.RWMutex
	personas    map[string]*Persona // slug -> persona
	bindings    map[string]string   // channelID -> slug
	configPath  string
	defaultPrompt string // config.yaml 的 default_system_prompt（无绑定时用）
}

func NewManager(configPath, defaultPrompt string) (*Manager, error) {
	m := &Manager{
		configPath:    configPath,
		defaultPrompt: defaultPrompt,
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) reload() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("read persona config: %w", err)
	}

	var cfg PersonaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse persona config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.personas = make(map[string]*Persona)
	for slug, p := range cfg.Personas {
		persona := &Persona{
			Name:     p.Name,
			Slug:     slug,
			SkillDir: p.SkillDir,
			Skills:   p.Skills,
		}
		if persona.SkillDir != "" {
			if err := persona.LoadPrompt(); err != nil {
				slog.Warn("failed to load persona prompt", "slug", slug, "err", err)
			}
		}
		m.personas[slug] = persona
	}
	m.bindings = cfg.Bindings
	return nil
}

// GetForChannel 根据频道 ID 获取角色 Prompt。无绑定时返回默认 prompt。
func (m *Manager) GetForChannel(channelID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	slug, ok := m.bindings[channelID]
	if !ok {
		return m.defaultPrompt
	}

	p, ok := m.personas[slug]
	if !ok || p.Prompt == "" {
		return m.defaultPrompt
	}
	return p.Prompt
}

// GetPersona 根据 slug 获取完整角色信息（命令用）。
func (m *Manager) GetPersona(slug string) (*Persona, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.personas[slug]
	return p, ok
}

// List 返回所有可用角色名称。
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.personas))
	for _, p := range m.personas {
		names = append(names, fmt.Sprintf("%s (%s)", p.Name, p.Slug))
	}
	return names
}

// Reload 重新加载配置文件。
func (m *Manager) Reload() error {
	return m.reload()
}
```

- [ ] Compile + test: `go build ./internal/persona/`

---

### Task 3: config + 配置文件

**Modify:** `internal/core/config.go` — 无改动（persona 配置路径单独给 Manager）

**Modify:** `config.example.yaml` — 加 persona 段引用

```yaml
persona:
  config_path: "./data/personas.yaml"
```

**Create:** `data/personas.yaml`

```yaml
personas:
  lin:
    name: "林"
    skill_dir: "data/personas/lin"
    skills: []

  default:
    name: "助手"
    skill_dir: ""
    skills: []

bindings: {}  # P2 先空着，用 /角色 切换 命令绑定
```

---

### Task 4: integrate into bot.go

**Modify:** `internal/core/bot.go`

改动点：

1. `Bot` 结构体加 `persona *persona.Manager`
2. `NewBot` 初始化 PersonaManager
3. `processMessage` 里替换 system prompt：

```go
// processMessage 中
systemPrompt := b.persona.GetForChannel(m.ChannelID)

// 命令路由：以 / 开头走命令
if strings.HasPrefix(text, "/") {
    go b.handleCommand(...)
    return
}

messages := b.llm.BuildMessages(systemPrompt, chatMsgs, text, nil)
```

---

### Task 5: 命令路由 + 角色管理命令

**Modify:** `internal/core/bot.go`

新增 `handleCommand` 函数：

```go
func (b *Bot) handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
    args := strings.Fields(m.Content)
    if len(args) == 0 {
        return
    }
    cmd := args[0]

    var reply string
    switch cmd {
    case "/角色":
        if len(args) < 2 {
            reply = "用法: /角色 列表|切换|信息|重载"
        } else {
            switch args[1] {
            case "列表":
                list := b.persona.List()
                reply = "已注册角色:\n" + strings.Join(list, "\n")
            case "切换":
                if len(args) < 3 {
                    reply = "用法: /角色 切换 <角色名>"
                } else {
                    // 更新 personas.yaml 的 bindings
                    // 然后 b.persona.Reload()
                    reply = fmt.Sprintf("当前频道已切换为 %s", args[2])
                }
            case "信息":
                // 查看当前频道绑定的角色详情
                slug := b.persona.GetCurrentSlug(m.ChannelID)
                p, ok := b.persona.GetPersona(slug)
                if ok {
                    reply = fmt.Sprintf("角色: %s\n%s", p.Name, truncate(p.Prompt, 300))
                }
            case "重载":
                b.persona.Reload()
                reply = "角色配置已重载"
            }
        }
    case "/help":
        reply = "可用命令:\n/角色 列表|切换|信息|重载\n/生成角色 名字 URL\n/角色校正 名字 内容"
    default:
        reply = "未知命令，输入 /help 查看"
    }

    s.ChannelMessageSendReply(m.ChannelID, reply, m.Reference())
}
```

---

### Task 6: 角色生成器

**Files:**
- Create: `internal/persona/generator/generator.go`
- Create: `internal/persona/generator/fetcher.go`
- Create: `internal/persona/generator/parser_bridge.go`
- Create: `internal/persona/generator/writer.go`

核心逻辑（设计文档已有完整代码）：

```
/生成角色 陈 https://prts.wiki/w/陈

内部流程:
  1. Go fetcher.SaveHTML → 抓取页面 HTML
  2. Go callParser → exec python3 tools/prts_parser.py → JSON
  3. JSON + prompts → DeepSeek 并行生成 4 层
  4. Go callWriter → exec python3 tools/character_skill_writer.py → SKILL.md
  5. 保存到 data/personas/chen/
  6. 写入 personas.yaml 绑定
  7. persona.Reload()
```

- [ ] 集成到 `/生成角色 名字 URL` 命令

---

### Task 7: 角色校正

**Files:**
- Create: `internal/persona/corrector.go`

```go
// /角色校正 林 林的语气应该更沉稳
func (m *Manager) Correct(slug string, instruction string) error {
    p, ok := m.GetPersona(slug)
    if !ok {
        return fmt.Errorf("角色 %s 不存在", slug)
    }
    
    // 读取 correction_handler.md 规则
    rule := readFile("prompts/correction_handler.md")
    
    // DeepSeek 按指令修正 persona.md
    corrected := llmClient.Chat(ctx, buildCorrectionMessages(rule, p, instruction))
    
    // 覆盖 persona.md → 重拼 SKILL.md → 热加载
    os.WriteFile(filepath.Join(p.SkillDir, "persona.md"), corrected, 0644)
    rebuildSkill(p.SkillDir)
    return m.Reload()
}
```

---

### P2 完成标准

- [ ] @机器人 对话使用 SKILL.md 人设（默认「林」）
- [ ] `/角色 列表` 显示所有角色
- [ ] `/角色 切换 林` 切换角色生效
- [ ] `/角色校正 林 xxx` AI 局部修正
- [ ] `/生成角色 陈 URL` 一条命令生成并可用
- [ ] 所有测试通过
