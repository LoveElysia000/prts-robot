// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 表示机器人的顶层配置结构。
type Config struct {
	QQ       QQConfig       `yaml:"qq"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

// QQConfig 表示 QQ 机器人配置。AppSecret 从环境变量 QQ_APP_SECRET 获取。
type QQConfig struct {
	AppID       string `yaml:"app_id"`
	AppSecret   string `yaml:"-"` // 仅从环境变量 QQ_APP_SECRET 注入
	WebhookPort int    `yaml:"webhook_port"`
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
// 支持的环境变量：QQ_APP_SECRET, DEEPSEEK_API_KEY, BOT_WEBHOOK_PORT。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.QQ.AppSecret = os.Getenv("QQ_APP_SECRET")
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.DeepSeek.APIKey = v
	}
	if v := os.Getenv("BOT_WEBHOOK_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err == nil {
			cfg.QQ.WebhookPort = port
		}
	}
	return cfg, nil
}
