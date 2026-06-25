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

// Manager 管理角色加载和频道绑定。
type Manager struct {
	mu            sync.RWMutex
	personas      map[string]*Persona
	bindings      map[string]string
	configPath    string
	defaultPrompt string
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

// Reload 重新加载 personas.yaml。
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

// GetPersona 按 slug 返回 Persona。
func (m *Manager) GetPersona(slug string) (*Persona, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.personas[slug]
	return p, ok
}

// List 返回所有角色名列表。
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.personas))
	for _, p := range m.personas {
		names = append(names, fmt.Sprintf("%s (%s)", p.Name, p.Slug))
	}
	return names
}
