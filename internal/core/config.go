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
	BotToken string `yaml:"-"`
}

type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`
	BaseURL             string `yaml:"base_url"`
	Model               string `yaml:"model"`
	DefaultSystemPrompt string `yaml:"default_system_prompt"`
}

type TriggerConfig struct {
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.Discord.BotToken = os.Getenv("DISCORD_BOT_TOKEN")
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.DeepSeek.APIKey = v
	}
	return cfg, nil
}
