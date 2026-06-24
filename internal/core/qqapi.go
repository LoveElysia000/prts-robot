// internal/core/qqapi.go
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type QQAPI struct {
	baseURL     string
	appID       string
	appSecret   string
	accessToken string
	mu          sync.RWMutex
}

func NewQQAPI(cfg QQConfig) *QQAPI {
	return &QQAPI{
		baseURL:   "https://api.sgroup.qq.com",
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
	}
}

func (q *QQAPI) EnsureToken() error {
	q.mu.RLock()
	if q.accessToken != "" {
		q.mu.RUnlock()
		return nil
	}
	q.mu.RUnlock()

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.accessToken != "" {
		return nil
	}

	token, err := q.getAccessToken()
	if err != nil {
		return err
	}
	q.accessToken = token

	go func() {
		time.Sleep(1*time.Hour + 50*time.Minute)
		q.mu.Lock()
		q.accessToken = ""
		q.mu.Unlock()
	}()

	return nil
}

func (q *QQAPI) getAccessToken() (string, error) {
	url := fmt.Sprintf("https://bots.qq.com/app/getAppAccessToken?appId=%s&clientSecret=%s", q.appID, q.appSecret)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access_token")
	}
	return result.AccessToken, nil
}

func (q *QQAPI) SendGroupMessage(groupID, content, replyMsgID string) error {
	q.mu.RLock()
	token := q.accessToken
	q.mu.RUnlock()

	body := map[string]any{
		"content":  content,
		"msg_type": 0,
		"msg_id":   replyMsgID,
	}
	data, _ := json.Marshal(body)

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
