// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// QQAPI 封装与 QQ 机器人开放平台的 HTTP API 交互。
// QQ Bot API v2 使用 AppSecret 直接在 Authorization 头鉴权。
type QQAPI struct {
	baseURL   string
	appSecret string
}

// NewQQAPI 创建 QQAPI 实例，使用 AppSecret 鉴权。
func NewQQAPI(appSecret string) *QQAPI {
	return &QQAPI{
		baseURL:   "https://api.sgroup.qq.com",
		appSecret: appSecret,
	}
}

// SendGroupMessage 向指定 QQ 群发送文本消息。
func (q *QQAPI) SendGroupMessage(groupID, content, replyMsgID string) error {
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
	req.Header.Set("Authorization", fmt.Sprintf("QQBot %s", q.appSecret))
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
