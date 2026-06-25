package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrompt(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# 测试角色\n这是一个测试"), 0644)

	p := &Persona{Slug: "test", SkillDir: dir}
	err := p.LoadPrompt()
	if err != nil {
		t.Fatalf("LoadPrompt failed: %v", err)
	}
	if p.Prompt != "# 测试角色\n这是一个测试" {
		t.Errorf("wrong prompt: %q", p.Prompt)
	}
}

func TestLoadPromptEmptyDir(t *testing.T) {
	p := &Persona{Slug: "empty", SkillDir: ""}
	err := p.LoadPrompt()
	if err != nil {
		t.Logf("expected error for empty dir: %v", err)
	}
}
