package generator

import (
	"context"
	"testing"

	"github.com/loveelysia000/robot/internal/llm"
)

func TestGenerateLayer_EmptyPrompt(t *testing.T) {
	g := &Generator{
		llm:    &llm.Client{},
		prompts: map[string]string{
			"persona_builder": "", // empty prompt
		},
		outputDir: t.TempDir(),
	}

	_, err := g.generateLayer(context.Background(), "persona_builder", `{"name":"test"}`, "test")
	if err == nil {
		t.Fatal("expected error for empty prompt, got nil")
	}
}

func TestGenerateLayer_MissingPrompt(t *testing.T) {
	g := &Generator{
		llm:      &llm.Client{},
		prompts:  map[string]string{},
		outputDir: t.TempDir(),
	}

	_, err := g.generateLayer(context.Background(), "nonexistent", `{}`, "test")
	if err == nil {
		t.Fatal("expected error for missing prompt, got nil")
	}
}
