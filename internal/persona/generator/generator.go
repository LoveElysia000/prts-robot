package generator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/loveelysia000/robot/internal/llm"
)

type Generator struct {
	llm       *llm.Client
	fetcher   *Fetcher
	prompts   map[string]string
	outputDir string
}

func NewGenerator(llmClient *llm.Client) *Generator {
	return &Generator{
		llm:       llmClient,
		fetcher:   NewFetcher(),
		prompts:   loadPrompts(),
		outputDir: "data/personas",
	}
}

func loadPrompts() map[string]string {
	names := []string{
		"persona_builder", "lore_builder",
		"relationship_builder", "custom_builder",
	}
	m := make(map[string]string)
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join("prompts", n+".md"))
		if err != nil {
			slog.Warn("prompt file missing, persona generation may be incomplete", "file", n+".md", "err", err)
			m[n] = ""
			continue
		}
		m[n] = string(data)
	}
	return m
}

type GenerateRequest struct {
	Slug    string
	Name    string
	WikiURL string
}

func (g *Generator) Generate(ctx context.Context, req GenerateRequest) error {
	htmlPath, err := g.fetcher.SaveHTML(ctx, req.WikiURL, req.Slug)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	profileJSON, err := runParser(htmlPath, req.Slug)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// 最多 2 个 LLM 请求并发，避免瞬间打满配额
	sem := make(chan struct{}, 2)

	var wg sync.WaitGroup
	layers := map[string]*string{
		"persona_builder":      new(string),
		"lore_builder":         new(string),
		"relationship_builder": new(string),
		"custom_builder":       new(string),
	}
	wg.Add(4)
	for key, ptr := range layers {
		go func(k string, p *string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				slog.Warn("generate layer skipped, context cancelled", "layer", k, "err", ctx.Err())
				return
			}
			result, err := g.generateLayer(ctx, k, profileJSON, req.Name)
			if err != nil {
				slog.Error("generate layer failed", "layer", k, "err", err)
			}
			*p = result
		}(key, ptr)
	}
	// 等待 4 个 LLM 调用全部完成
	wg.Wait()

	if *layers["persona_builder"] == "" {
		return fmt.Errorf("persona_builder layer produced no output, generation aborted")
	}

	dir := filepath.Join(g.outputDir, req.Slug)
	os.MkdirAll(dir, 0755)
	writeFile := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			slog.Error("write persona file failed", "file", name, "err", err)
		}
	}
	writeFile("persona.md", *layers["persona_builder"])
	writeFile("lore.md", *layers["lore_builder"])
	writeFile("relationship.md", *layers["relationship_builder"])
	writeFile("custom.md", *layers["custom_builder"])

	if err := RunWriter(req.Slug, req.Name, req.WikiURL, dir); err != nil {
		return fmt.Errorf("writer: %w", err)
	}
	return nil
}

func (g *Generator) generateLayer(ctx context.Context, ruleName, profileJSON, name string) (string, error) {
	rule := g.prompts[ruleName]
	if rule == "" {
		return "", fmt.Errorf("prompt %q is empty or missing", ruleName)
	}
	messages := g.llm.BuildMessages(rule, nil,
		fmt.Sprintf("角色名: %s\n\n解析结果:\n%s", name, profileJSON), nil)
	return g.llm.Chat(ctx, messages)
}
