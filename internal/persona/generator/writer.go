package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runParser(htmlPath, slug string) (string, error) {
	outputPath := filepath.Join(os.TempDir(), "prts_"+slug+".json")
	cmd := exec.Command("python3",
		"tools/prts_parser.py",
		"--input", htmlPath,
		"--output", outputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("parser failed: %s: %w", string(out), err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func RunWriter(slug, name, sourceURL, dir string) error {
	cmd := exec.Command("python3",
		"tools/character_skill_writer.py",
		"--action", "create",
		"--slug", slug,
		"--name", name,
		"--source-url", sourceURL,
		"--persona", filepath.Join(dir, "persona.md"),
		"--lore", filepath.Join(dir, "lore.md"),
		"--relationship", filepath.Join(dir, "relationship.md"),
		"--custom", filepath.Join(dir, "custom.md"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("writer failed: %s: %w", string(out), err)
	}
	return nil
}
