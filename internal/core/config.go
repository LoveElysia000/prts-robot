// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 表示机器人的顶层配置结构。
type Config struct {
	NapCat   NapCatConfig   `yaml:"napcat"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

// NapCatConfig 表示 NapCat 配置。
type NapCatConfig struct {
	AccessToken string `yaml:"access_token"`
}

// DeepSeekConfig 表示 DeepSeek 配置，API 密钥通过环境变量注入。
type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`
	BaseURL             string `yaml:"base_url"`
	Model               string `yaml:"model"`
	DefaultSystemPrompt string `yaml:"default_system_prompt"`
}

// TriggerConfig 表示消息触发方式配置。
type TriggerConfig struct {
	Mode          string `yaml:"mode"`
	CommandPrefix string `yaml:"command_prefix"`
}

// DatabaseConfig 表示数据库配置。
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// LoadConfig 从 YAML 文件加载配置，然后通过环境变量覆盖敏感字段。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.DeepSeek.APIKey = v
	}
	return cfg, nil
}
