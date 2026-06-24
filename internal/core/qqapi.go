// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// QQAPI 封装与 QQ 机器人开放平台的 HTTP API 交互。
// 使用 POST /app/getAppAccessToken 获取 token，Bearer 方式鉴权。
type QQAPI struct {
	baseURL     string
	appID       string
	appSecret   string
	accessToken string
	expiresAt   time.Time
	mu          sync.RWMutex
}

// NewQQAPI 创建 QQAPI 实例。
func NewQQAPI(appID, appSecret string) *QQAPI {
	return &QQAPI{
		baseURL:   "https://api.sgroup.qq.com",
		appID:     appID,
		appSecret: appSecret,
	}
}

// refreshToken 从 QQ 平台获取 access_token（POST JSON Body 方式）。
func (q *QQAPI) refreshToken() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	body := map[string]string{
		"appId":        q.appID,
		"clientSecret": q.appSecret,
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(
		"https://bots.qq.com/app/getAppAccessToken",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("get token failed: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode token: %w", err)
	}

	q.accessToken = result.AccessToken
	q.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return nil
}

// ensureToken 确保 access_token 有效，过期自动刷新。
func (q *QQAPI) ensureToken() error {
	q.mu.RLock()
	valid := q.accessToken != "" && time.Now().Before(q.expiresAt)
	q.mu.RUnlock()
	if valid {
		return nil
	}
	return q.refreshToken()
}

// SendGroupMessage 向指定 QQ 群发送文本消息。
func (q *QQAPI) SendGroupMessage(groupID, content, replyMsgID string) error {
	if err := q.ensureToken(); err != nil {
		return fmt.Errorf("ensure token: %w", err)
	}

	q.mu.RLock()
	token := q.accessToken
	q.mu.RUnlock()

	body := map[string]any{
		"content":  content,
		"msg_type": 0,
		"msg_id":   replyMsgID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/v2/groups/%s/messages", q.baseURL, groupID),
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("QQBot %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send message failed: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
