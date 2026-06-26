package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runParser 调用 Python 脚本 tools/prts_parser.py 将抓取的 wiki HTML 解析为结构化 JSON。
// 输出写入临时文件，读取后返回 JSON 字符串。
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
