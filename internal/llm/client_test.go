// Package llm 封装与大语言模型（DeepSeek）的交互，提供对话消息构建和聊天完成功能。
package llm

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

// TestBuildMessages 验证 BuildMessages 能正确构建包含系统提示词、历史消息和用户输入的消息列表。
func TestBuildMessages(t *testing.T) {
	client := &Client{}
	sysPrompt := "你是一个助手"
	sessionMsgs := []ChatMessage{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好！"},
	}
	userText := "天气？"

	messages := client.BuildMessages(sysPrompt, sessionMsgs, userText, nil)

	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}
	if messages[0].Role != openai.ChatMessageRoleSystem {
		t.Error("first message should be system")
	}
}
