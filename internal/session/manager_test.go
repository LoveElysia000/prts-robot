// Package session 提供群会话管理功能，使用 SQLite 持久化存储对话历史消息。
package session

import (
	"database/sql"
	"os"
	"testing"
)

// setupDB 创建一个临时 SQLite 数据库用于测试，返回数据库实例。
func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	f, _ := os.CreateTemp("", "test-*.db")
	f.Close()
	db, err := sql.Open("sqlite", f.Name())
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	return db
}

// TestAppendAndGetRecent 验证 Append 和 GetRecent 能正确存储和按序获取会话消息。
func TestAppendAndGetRecent(t *testing.T) {
	db := setupDB(t)
	mgr, err := NewManager(db)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.Append("group_123", Message{Role: "user", Content: "你好"})
	mgr.Append("group_123", Message{Role: "assistant", Content: "你好！"})
	mgr.Append("group_123", Message{Role: "user", Content: "天气？"})
	mgr.Append("group_123", Message{Role: "assistant", Content: "28°C"})

	recent, err := mgr.GetRecent("group_123", 2)
	if err != nil {
		t.Fatalf("GetRecent failed: %v", err)
	}
	if len(recent) != 4 {
		t.Fatalf("expected 4 messages (2 rounds), got %d", len(recent))
	}
	if recent[0].Role != "user" || recent[0].Content != "你好" {
		t.Errorf("wrong first message: %+v", recent[0])
	}
}

// TestGetRecentEmpty 验证 GetRecent 在不存在的会话键上能正确返回空列表而不报错。
func TestGetRecentEmpty(t *testing.T) {
	db := setupDB(t)
	mgr, _ := NewManager(db)
	recent, _ := mgr.GetRecent("group_nonexist", 10)
	if len(recent) != 0 {
		t.Errorf("expected 0 messages, got %d", len(recent))
	}
}
