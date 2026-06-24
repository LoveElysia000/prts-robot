// Package session 提供群会话管理功能，使用 SQLite 持久化存储对话历史消息。
package session

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// Message 表示一条会话消息记录，包含角色、内容和创建时间。
type Message struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// Manager 管理群会话消息的读写，封装了 SQLite 数据库操作。
type Manager struct {
	db *sql.DB
}

// NewManager 创建新的会话管理器，自动初始化数据库中的消息表。
func NewManager(db *sql.DB) (*Manager, error) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_key TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_key, id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return nil, err
		}
	}
	return &Manager{db: db}, nil
}

// Append 将会话消息追加到数据库，按会话键值存储。
func (m *Manager) Append(sessionKey string, msg Message) error {
	_, err := m.db.Exec(
		`INSERT INTO messages (session_key, role, content) VALUES (?, ?, ?)`,
		sessionKey, msg.Role, msg.Content,
	)
	return err
}

// GetRecent 从数据库中获取指定会话最近的若干轮对话消息，按时间正序返回。
func (m *Manager) GetRecent(sessionKey string, rounds int) ([]Message, error) {
	rows, err := m.db.Query(
		`SELECT role, content, created_at FROM messages
		 WHERE session_key = ?
		 ORDER BY id DESC LIMIT ?`,
		sessionKey, rounds*2,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}
