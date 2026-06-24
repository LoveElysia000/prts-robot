// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// QQAPI 封装与 QQ 机器人开放平台的 HTTP API 交互，包括 access_token 获取和群消息发送。
// access_token 有两小时有效期，会自动刷新。
type QQAPI struct {
	baseURL     string
	appID       string
	appSecret   string
	accessToken string
	mu          sync.RWMutex
	refreshing  atomic.Bool // 防止并发重复刷新 token
}

// NewQQAPI 根据 QQConfig 创建一个新的 QQAPI 实例。
func NewQQAPI(cfg QQConfig) *QQAPI {
	return &QQAPI{
		baseURL:   "https://api.sgroup.qq.com",
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
	}
}

// EnsureToken 检查当前 access_token 是否有效，若无效则重新获取。
func (q *QQAPI) EnsureToken() error {
	q.mu.RLock()
	if q.accessToken != "" {
		q.mu.RUnlock()
		return nil
	}
	q.mu.RUnlock()

	return q.refreshToken()
}

// refreshToken 获取新的 access_token 并设置自动刷新定时器。
func (q *QQAPI) refreshToken() error {
	// 防止并发重复刷新
	if !q.refreshing.CompareAndSwap(false, true) {
		return nil
	}
	defer q.refreshing.Store(false)

	q.mu.Lock()
	defer q.mu.Unlock()

	// double-check
	if q.accessToken != "" {
		return nil
	}

	token, expiresIn, err := q.getAccessToken()
	if err != nil {
		return err
	}
	q.accessToken = token

	// 提前 5 分钟刷新
	refreshAfter := time.Duration(expiresIn-300) * time.Second
	if refreshAfter <= 0 {
		refreshAfter = time.Hour
	}
	time.AfterFunc(refreshAfter, func() {
		q.mu.Lock()
		q.accessToken = ""
		q.mu.Unlock()
	})

	return nil
}

// getAccessToken 向 QQ 开放平台请求新的 access_token。
func (q *QQAPI) getAccessToken() (token string, expiresIn int, err error) {
	resp, err := http.Post(
		fmt.Sprintf("%s/app/getAppAccessToken?appId=%s&clientSecret=%s", q.baseURL, q.appID, q.appSecret),
		"application/json",
		nil,
	)
	if err != nil {
		return "", 0, fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("get token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, fmt.Errorf("decode token: %w", err)
	}
	if result.AccessToken == "" {
		return "", 0, fmt.Errorf("empty access_token in response")
	}
	return result.AccessToken, result.ExpiresIn, nil
}

// SendGroupMessage 向指定 QQ 群发送文本消息，可选择引用回复某条消息。
// 如果 access_token 已过期，会自动刷新后重试。
func (q *QQAPI) SendGroupMessage(groupID, content, replyMsgID string) error {
	q.mu.RLock()
	token := q.accessToken
	q.mu.RUnlock()

	if token == "" {
		if err := q.refreshToken(); err != nil {
			return fmt.Errorf("refresh token: %w", err)
		}
		q.mu.RLock()
		token = q.accessToken
		q.mu.RUnlock()
	}

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
		return fmt.Errorf("send message failed: status %d", resp.StatusCode)
	}
	return nil
}
