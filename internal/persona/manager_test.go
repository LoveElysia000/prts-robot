package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("lin persona prompt"), 0644)
	configContent := `
personas:
  lin:
    name: "林"
    skill_dir: "` + dir + `"
    skills: []
  default:
    name: "助手"
    skill_dir: ""
    skills: []
bindings:
  "channel_123": "lin"
`
	f, _ := os.CreateTemp("", "personas-*.yaml")
	defer os.Remove(f.Name())
	f.WriteString(configContent)
	f.Close()

	mgr, err := NewManager(f.Name(), "默认prompt")
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	p, ok := mgr.GetPersona("lin")
	if !ok {
		t.Fatal("lin persona not found")
	}
	if p.Name != "林" {
		t.Errorf("expected 林, got %s", p.Name)
	}
	if p.Prompt != "lin persona prompt" {
		t.Errorf("wrong prompt: %q", p.Prompt)
	}
}

func TestGetForChannel(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("lin persona prompt"), 0644)
	configContent := `
personas:
  lin:
    name: "林"
    skill_dir: "` + dir + `"
    skills: []
  default:
    name: "助手"
    skill_dir: ""
    skills: []
bindings:
  "channel_123": "lin"
`
	f, _ := os.CreateTemp("", "personas-*.yaml")
	defer os.Remove(f.Name())
	f.WriteString(configContent)
	f.Close()

	mgr, err := NewManager(f.Name(), "默认prompt")
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	prompt := mgr.GetForChannel("channel_123")
	if prompt == "默认prompt" {
		t.Error("bound channel should get persona prompt, not default")
	}
	if prompt != "lin persona prompt" {
		t.Errorf("expected 'lin persona prompt', got %q", prompt)
	}

	prompt = mgr.GetForChannel("no_binding")
	if prompt != "默认prompt" {
		t.Error("unbound channel should get default prompt")
	}
}

func TestList(t *testing.T) {
	configContent := `
personas:
  lin:
    name: "林"
    skill_dir: ""
    skills: []
bindings: {}
`
	f, _ := os.CreateTemp("", "personas-*.yaml")
	defer os.Remove(f.Name())
	f.WriteString(configContent)
	f.Close()

	mgr, _ := NewManager(f.Name(), "默认")
	list := mgr.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 persona, got %d", len(list))
	}
	if list[0].Name != "林" || list[0].Slug != "lin" {
		t.Errorf("expected Name=林 Slug=lin, got Name=%q Slug=%q", list[0].Name, list[0].Slug)
	}
}
