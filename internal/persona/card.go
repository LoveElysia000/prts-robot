package persona

import (
	"fmt"
	"os"
	"path/filepath"
)

// Persona 表示一个角色。
type Persona struct {
	Name     string   // 角色名
	Slug     string   // 标识，如 "lin"
	SkillDir string   // SKILL.md 所在目录
	Prompt   string   // SKILL.md 内容
	Skills   []string // Function 工具名（P3 用）
}

// LoadPrompt 从 SkillDir 读取 SKILL.md 并填充 Prompt。
func (p *Persona) LoadPrompt() error {
	if p.SkillDir == "" {
		return fmt.Errorf("persona %s: empty SkillDir", p.Slug)
	}
	data, err := os.ReadFile(filepath.Join(p.SkillDir, "SKILL.md"))
	if err != nil {
		return fmt.Errorf("persona %s: %w", p.Slug, err)
	}
	p.Prompt = string(data)
	return nil
}
