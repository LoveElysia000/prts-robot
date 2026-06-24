package core

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := `
napcat:
  access_token: "test-token"
deepseek:
  api_key: "sk-test"
  base_url: "https://api.deepseek.com"
  model: "deepseek-v4-flash"
  default_system_prompt: "test"
trigger:
  mode: "hybrid"
  command_prefix: "/"
database:
  path: "./data/bot.db"
`
	f, _ := os.CreateTemp("", "config-*.yaml")
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.NapCat.AccessToken != "test-token" {
		t.Errorf("expected test-token, got %s", cfg.NapCat.AccessToken)
	}
	if cfg.DeepSeek.Model != "deepseek-v4-flash" {
		t.Errorf("expected deepseek-v4-flash, got %s", cfg.DeepSeek.Model)
	}
}
