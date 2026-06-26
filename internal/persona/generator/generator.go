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
		data, _ := os.ReadFile(filepath.Join("prompts", n+".md"))
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
	htmlPath, err := g.fetcher.SaveHTML(req.WikiURL, req.Slug)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	profileJSON, err := runParser(htmlPath, req.Slug)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

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
			result, err := g.generateLayer(ctx, k, profileJSON, req.Name)
			if err != nil {
				slog.Error("generate layer failed", "layer", k, "err", err)
			}
			*p = result
		}(key, ptr)
	}
	wg.Wait()

	dir := filepath.Join(g.outputDir, req.Slug)
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "persona.md"), []byte(*layers["persona_builder"]), 0644)
	os.WriteFile(filepath.Join(dir, "lore.md"), []byte(*layers["lore_builder"]), 0644)
	os.WriteFile(filepath.Join(dir, "relationship.md"), []byte(*layers["relationship_builder"]), 0644)
	os.WriteFile(filepath.Join(dir, "custom.md"), []byte(*layers["custom_builder"]), 0644)

	if err := RunWriter(req.Slug, req.Name, req.WikiURL, dir); err != nil {
		return fmt.Errorf("writer: %w", err)
	}
	return nil
}

func (g *Generator) generateLayer(ctx context.Context, ruleName, profileJSON, name string) (string, error) {
	rule := g.prompts[ruleName]
	messages := g.llm.BuildMessages(rule, nil,
		fmt.Sprintf("角色名: %s\n\n解析结果:\n%s", name, profileJSON), nil)
	return g.llm.Chat(ctx, messages)
}
