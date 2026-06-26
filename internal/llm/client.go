// Package llm 封装与大语言模型（DeepSeek）的交互，提供对话消息构建和聊天完成功能。
package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// ChatMessage 表示一条对话消息，包含角色和内容。
type ChatMessage struct {
	Role    string
	Content string
}

// Client 封装与 OpenAI 兼容 API（DeepSeek）的客户端，使用 go-openai 库进行通信。
type Client struct {
	api   *openai.Client
	model string
}

// DeepSeekConfig 表示 DeepSeek 客户端的配置参数，包括 API 密钥、基础 URL 和模型名称。
type DeepSeekConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewClient 根据 DeepSeekConfig 创建一个新的 LLM 客户端实例。
func NewClient(cfg DeepSeekConfig) *Client {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL
	api := openai.NewClientWithConfig(config)
	return &Client{api: api, model: cfg.Model}
}

// Chat 发送聊天完成请求到 DeepSeek API，返回模型生成的回复内容。
func (c *Client) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("deepseek api error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return resp.Choices[0].Message.Content, nil
}

// BuildMessages 构建发送给模型的聊天消息列表。
// 顺序：system prompt → 历史消息(多轮 user/assistant) → 当前用户输入。
func (c *Client) BuildMessages(systemPrompt string, sessionMsgs []ChatMessage, userText string, tools []openai.Tool) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, 1+len(sessionMsgs)+1)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	for _, m := range sessionMsgs {
		role := openai.ChatMessageRoleUser
		if m.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userText,
	})
	return messages
}
