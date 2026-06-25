package generator

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// RunWriter 调用 Python character_skill_writer.py 拼装 SKILL.md。
func RunWriter(slug, name, sourceURL, dir string) error {
	cmd := exec.Command("python3",
		"tools/character_skill_writer.py",
		"--action", "create",
		"--slug", slug,
		"--name", name,
		"--source-url", sourceURL,
		"--base-dir", filepath.Dir(dir),
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
