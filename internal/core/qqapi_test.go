// internal/core/qqapi_test.go
package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
