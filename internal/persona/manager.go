package persona

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// PersonaConfig 对应 personas.yaml 的结构体。
type PersonaConfig struct {
	Personas map[string]struct {
		Name     string   `yaml:"name"`
		SkillDir string   `yaml:"skill_dir"`
		Skills   []string `yaml:"skills"`
	} `yaml:"personas"`
	Bindings map[string]string `yaml:"bindings"`
}

// Manager 管理角色加载和频道绑定，并发安全（RWMutex）。
type Manager struct {
	mu            sync.RWMutex
	personas      map[string]*Persona  // slug → 角色对象
	bindings      map[string]string    // channelID → slug
	configPath    string               // personas.yaml 路径
	defaultPrompt string               // 无绑定或加载失败时的回退 prompt
}

// NewManager 从配置文件加载角色，defaultPrompt 是无绑定时使用的 prompt。
func NewManager(configPath, defaultPrompt string) (*Manager, error) {
	m := &Manager{
		configPath:    configPath,
		defaultPrompt: defaultPrompt,
	}
	if err := m.Reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// Reload 重新加载 personas.yaml，原子替换内存中的角色和绑定数据。
// 持写锁，调用期间所有 GetForChannel/FindPersona/List 会等待。
func (m *Manager) Reload() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("read personas config: %w", err)
	}
	var cfg PersonaConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse personas config: %w", err)
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

// clonePersona 返回 Persona 的深拷贝，防止调用方通过指针修改内部状态。
func clonePersona(p *Persona) *Persona {
	cp := &Persona{
		Name:     p.Name,
		Slug:     p.Slug,
		SkillDir: p.SkillDir,
		Prompt:   p.Prompt,
	}
	if p.Skills != nil {
		cp.Skills = append([]string(nil), p.Skills...)
	}
	return cp
}

// GetForChannel 返回频道绑定的角色 prompt。无绑定返回默认值。
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

// GetPersona 按 slug 返回 Persona 的拷贝。
func (m *Manager) GetPersona(slug string) (*Persona, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.personas[slug]
	if !ok {
		return nil, false
	}
	return clonePersona(p), true
}

// FindPersona 按名字或 slug 查找角色。
func (m *Manager) FindPersona(query string) (*Persona, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.personas[query]; ok {
		return clonePersona(p), true
	}
	for _, p := range m.personas {
		if p.Name == query {
			return clonePersona(p), true
		}
	}
	return nil, false
}

// List 返回所有已注册角色的拷贝。
func (m *Manager) List() []*Persona {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Persona, 0, len(m.personas))
	for _, p := range m.personas {
		list = append(list, clonePersona(p))
	}
	return list
}
