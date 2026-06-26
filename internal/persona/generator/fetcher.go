package generator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Fetcher struct {
	client *http.Client
}

func NewFetcher() *Fetcher {
	// 抓取 wiki 页面时不走代理（走代理可能卡住或被墙），30s 超时防止永久挂起
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
			},
		},
	}
}

func (f *Fetcher) SaveHTML(ctx context.Context, url, slug string) (string, error) {
	// 使用 ctx 支持上游取消（如生成超时），Client.Timeout 作为兜底
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := f.client.Do(req)
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
