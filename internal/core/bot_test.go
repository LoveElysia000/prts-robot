package core

import (
	"testing"

	"github.com/sashabaranov/go-openai"

	"github.com/loveelysia000/robot/internal/llm"
	"github.com/loveelysia000/robot/internal/persona"
	"github.com/loveelysia000/robot/internal/session"
)

func TestBuildMessages(t *testing.T) {
	llmClient := &llm.Client{}

	tests := []struct {
		name         string
		systemPrompt string
		history      []session.Message
		text         string
		wantLen      int
		wantLastRole string
	}{
		{
			name:         "empty history",
			systemPrompt: "你是一个助手",
			history:      nil,
			text:         "你好",
			wantLen:      2,
			wantLastRole: openai.ChatMessageRoleUser,
		},
		{
			name:         "with history",
			systemPrompt: "你是一个助手",
			history: []session.Message{
				{Role: "user", Content: "你好"},
				{Role: "assistant", Content: "你好！"},
			},
			text:         "天气？",
			wantLen:      4,
			wantLastRole: openai.ChatMessageRoleUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot := &Bot{llm: llmClient}
			msgs := bot.buildMessages(tt.systemPrompt, tt.history, tt.text)
			if len(msgs) != tt.wantLen {
				t.Errorf("expected %d messages, got %d", tt.wantLen, len(msgs))
			}
			if msgs[0].Role != openai.ChatMessageRoleSystem {
				t.Error("first message should be system")
			}
			if msgs[len(msgs)-1].Role != tt.wantLastRole {
				t.Errorf("last message role = %q, want %q", msgs[len(msgs)-1].Role, tt.wantLastRole)
			}
			if msgs[len(msgs)-1].Content != tt.text {
				t.Errorf("last message content = %q, want %q", msgs[len(msgs)-1].Content, tt.text)
			}
		})
	}
}

func TestFormatPersonaList(t *testing.T) {
	list := []*persona.Persona{
		{Name: "林", Slug: "lin"},
		{Name: "助手", Slug: "default"},
	}
	got := formatPersonaList(list)
	want := "林 (lin)\n助手 (default)"
	if got != want {
		t.Errorf("formatPersonaList:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatPersonaListEmpty(t *testing.T) {
	got := formatPersonaList(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

