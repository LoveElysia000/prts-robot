package generator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Fetcher struct{}

func (f *Fetcher) SaveHTML(url, slug string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	dir := filepath.Join("data", "personas", slug)
	os.MkdirAll(dir, 0755)
	htmlPath := filepath.Join(dir, "page.html")

	file, err := os.Create(htmlPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	io.Copy(file, resp.Body)
	return htmlPath, nil
}
