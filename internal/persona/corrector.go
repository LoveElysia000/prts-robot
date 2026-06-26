package persona

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/persona/generator"
)

// Correct 用 AI 按指令修正角色的 persona.md，然后重新拼装 SKILL.md。
// 流程：读 correction_handler.md 规则 → 读角色当前 persona.md →
// 发给 LLM 修正 → 写回 persona.md → 重新生成 SKILL.md。
func (m *Manager) Correct(ctx context.Context, llmClient *llm.Client, slug, instruction string) error {
	p, ok := m.GetPersona(slug)
	if !ok {
		return fmt.Errorf("角色 %s 不存在", slug)
	}

	// correction_handler.md 定义了 LLM 修正 persona 时遵循的规则
	rule, err := os.ReadFile("prompts/correction_handler.md")
	if err != nil {
		return fmt.Errorf("read correction rule: %w", err)
	}

	personaContent, err := os.ReadFile(filepath.Join(p.SkillDir, "persona.md"))
	if err != nil {
		return fmt.Errorf("read persona.md: %w", err)
	}

	messages := llmClient.BuildMessages(string(rule), nil,
		fmt.Sprintf("校正指令: %s\n\n当前 persona:\n%s", instruction, string(personaContent)), nil)

	corrected, err := llmClient.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("llm correction: %w", err)
	}

	if err := os.WriteFile(filepath.Join(p.SkillDir, "persona.md"), []byte(corrected), 0644); err != nil {
		return fmt.Errorf("write persona.md: %w", err)
	}

	return generator.RunWriter(p.Slug, p.Name, "", p.SkillDir)
}
