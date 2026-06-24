// Package message 定义消息结构和消息处理逻辑，包括消息类型、触发判断和命令解析。
package message

import "strings"

// Message 表示一条聊天消息，支持群聊和私聊。
type Message struct {
	GroupID string
	UserID  string
	Text    string
	IsAtBot bool
	MsgID   string
}

// IsCommand 判断消息文本是否以指定前缀开头。
func (m *Message) IsCommand(prefix string) bool {
	return strings.HasPrefix(m.Text, prefix)
}

// SessionKey 返回该消息对应的会话键值。群聊用 group_{id}，私聊用 private_{id}。
func (m *Message) SessionKey() string {
	if m.GroupID != "" {
		return "group_" + m.GroupID
	}
	return "private_" + m.UserID
}

// IsPrivate 判断是否为私聊消息。
func (m *Message) IsPrivate() bool {
	return m.GroupID == ""
}
