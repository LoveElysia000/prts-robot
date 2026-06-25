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
