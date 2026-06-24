// Package core 提供机器人核心功能的单元测试。
package core

import (
	"os"
	"testing"
)

// TestLoadConfig 验证 YAML 配置加载和环境变量覆盖。
func TestLoadConfig(t *testing.T) {
	content := `
qq:
  app_id: "123"
  webhook_port: 8080
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
	if cfg.QQ.AppID != "123" {
		t.Errorf("expected app_id 123, got %s", cfg.QQ.AppID)
	}
	if cfg.DeepSeek.Model != "deepseek-v4-flash" {
		t.Errorf("expected model deepseek-v4-flash, got %s", cfg.DeepSeek.Model)
	}
	if cfg.Trigger.Mode != "hybrid" {
		t.Errorf("expected mode hybrid, got %s", cfg.Trigger.Mode)
	}
}
