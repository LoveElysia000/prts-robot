// internal/core/config.go
package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	QQ       QQConfig       `yaml:"qq"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

type QQConfig struct {
	AppID       string `yaml:"app_id"`
	AppSecret   string `yaml:"app_secret"`
	WebhookPort int    `yaml:"webhook_port"`
}

type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`
	BaseURL             string `yaml:"base_url"`
	Model               string `yaml:"model"`
	DefaultSystemPrompt string `yaml:"default_system_prompt"`
}

type TriggerConfig struct {
	Mode          string `yaml:"mode"`
	CommandPrefix string `yaml:"command_prefix"`
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
	return cfg, nil
}
