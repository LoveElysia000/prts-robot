// internal/llm/client.go
package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type ChatMessage struct {
	Role    string
	Content string
}

type Client struct {
	api   *openai.Client
	model string
}

type DeepSeekConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewClient(cfg DeepSeekConfig) *Client {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL
	api := openai.NewClientWithConfig(config)
	return &Client{api: api, model: cfg.Model}
}

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
