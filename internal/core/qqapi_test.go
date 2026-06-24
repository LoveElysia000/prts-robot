// Package core 提供机器人核心功能，包括配置加载、QQ API 交互和 webhook 处理。
package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSendGroupMessage 验证 SendGroupMessage 能正确携带认证头并发送 POST 请求到 QQ API。
func TestSendGroupMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Error("expected POST")
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing auth header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123"}`))
	}))
	defer server.Close()

	api := &QQAPI{
		baseURL:     server.URL,
		accessToken: "test-token",
	}
	err := api.SendGroupMessage("group_123", "你好", "")
	if err != nil {
		t.Fatalf("SendGroupMessage failed: %v", err)
	}
}
