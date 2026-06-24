// Package core 提供机器人核心功能的单元测试。
package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSendGroupMessage 验证群消息发送功能。
func TestSendGroupMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("missing auth header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	api := &QQAPI{
		baseURL:     server.URL,
		accessToken: "test-token",
		expiresAt:   time.Now().Add(time.Hour),
	}
	err := api.SendGroupMessage("group_123", "你好", "")
	if err != nil {
		t.Fatalf("SendGroupMessage failed: %v", err)
	}
}
