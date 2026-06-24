// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 表示机器人的顶层配置结构，包含 QQ、DeepSeek、触发方式和数据库配置。
type Config struct {
	QQ       QQConfig       `yaml:"qq"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Trigger  TriggerConfig  `yaml:"trigger"`
	Database DatabaseConfig `yaml:"database"`
}

// QQConfig 表示 QQ 机器人配置，包含应用 ID（配置）、应用密钥（环境变量）和 webhook 端口。
type QQConfig struct {
	AppID       string `yaml:"app_id"`
	AppSecret   string `yaml:"app_secret"`   // 文件中留空，通过环境变量 QQ_APP_SECRET 注入
	WebhookPort int    `yaml:"webhook_port"`
}

// DeepSeekConfig 表示 DeepSeek 大语言模型配置，API 密钥通过环境变量注入。
type DeepSeekConfig struct {
	APIKey              string `yaml:"api_key"`                // 文件中留空，通过环境变量 DEEPSEEK_API_KEY 注入
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

// LoadConfig 从指定路径读取 YAML 配置文件，再通过环境变量覆盖敏感字段后返回 Config。
// 支持的环境变量：
//   - QQ_APP_SECRET: app_secret 私钥
//   - DEEPSEEK_API_KEY: DeepSeek API 密钥
//   - QQ_WEBHOOK_PORT: 覆盖 webhook 监听端口
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// 环境变量覆盖敏感字段
	if v := os.Getenv("QQ_APP_SECRET"); v != "" {
		cfg.QQ.AppSecret = v
	}
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.DeepSeek.APIKey = v
	}
	if v := os.Getenv("QQ_WEBHOOK_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err == nil {
			cfg.QQ.WebhookPort = port
		}
	}
	return cfg, nil
}
