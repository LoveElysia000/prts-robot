// Package core 提供机器人核心功能，包括配置加载、Discord 连接和消息处理。
package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Discord  DiscordConfig  `yaml:"discord"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

type DiscordConfig struct {
	// BotToken 不从 yaml 读取，由环境变量 DISCORD_BOT_TOKEN 注入
	BotToken string `yaml:"-"`
}

type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`                // 也可通过环境变量 DEEPSEEK_API_KEY 覆盖
	BaseURL             string `yaml:"base_url"`
	Model               string `yaml:"model"`
	DefaultSystemPrompt string `yaml:"default_system_prompt"` // 无角色绑定时的默认 prompt
}

type TriggerConfig struct {
	Mode string `yaml:"mode"` // "mention" 仅@回复, "all" 所有消息
}

type DatabaseConfig struct {
	Path string `yaml:"path"` // SQLite 文件路径
}

// LoadConfig 从 yaml 文件加载配置，敏感字段（token/key）可从环境变量覆盖。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// 敏感信息优先从环境变量读取，yaml 中的值作为回退
	cfg.Discord.BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.DeepSeek.APIKey = v
	}
	return cfg, nil
}
