// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config 表示机器人的顶层配置结构，包含 QQ、DeepSeek、触发方式和数据库配置。
type Config struct {
	QQ       QQConfig       `yaml:"qq"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

// QQConfig 表示 QQ 机器人配置，包含应用 ID、应用密钥和 webhook 端口。
type QQConfig struct {
	AppID       string `yaml:"app_id"`
	AppSecret   string `yaml:"app_secret"`
	WebhookPort int    `yaml:"webhook_port"`
}

// DeepSeekConfig 表示 DeepSeek 大语言模型配置，包含 API 密钥、基础 URL、模型名称和默认系统提示词。
type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`
	BaseURL             string `yaml:"base_url"`
	Model               string `yaml:"model"`
	DefaultSystemPrompt string `yaml:"default_system_prompt"`
}

// TriggerConfig 表示消息触发方式配置，包含触发模式和命令前缀。
type TriggerConfig struct {
	Mode          string `yaml:"mode"`
	CommandPrefix string `yaml:"command_prefix"`
}

// DatabaseConfig 表示数据库配置，包含数据库文件路径。
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// LoadConfig 从指定路径读取 YAML 配置文件并解析为 Config 结构体。
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
