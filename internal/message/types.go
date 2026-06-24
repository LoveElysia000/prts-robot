// internal/message/types.go
package message

import "strings"

type Message struct {
	GroupID string
	UserID  string
	Text    string
	IsAtBot bool
	MsgID   string
}

func (m *Message) IsCommand(prefix string) bool {
	return strings.HasPrefix(m.Text, prefix)
}

func (m *Message) SessionKey() string {
	return "group_" + m.GroupID
}
