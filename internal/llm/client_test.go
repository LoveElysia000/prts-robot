// internal/llm/client_test.go
package llm

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

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
