// internal/session/manager.go
package session

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Message struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) (*Manager, error) {
	query := `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_key TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_key, id);`

	if _, err := db.Exec(query); err != nil {
		return nil, err
	}
	return &Manager{db: db}, nil
}

func (m *Manager) Append(sessionKey string, msg Message) error {
	_, err := m.db.Exec(
		`INSERT INTO messages (session_key, role, content) VALUES (?, ?, ?)`,
		sessionKey, msg.Role, msg.Content,
	)
	return err
}

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
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}
