// Package message 定义消息结构和消息处理逻辑，包括消息类型、触发判断和命令解析。
package message

import "strings"

// Message 表示一条聊天消息，支持群聊和私聊。
type Message struct {
	ChannelID string
	UserID    string
	Text      string
	IsDM      bool
	IsAtBot   bool
}

// IsCommand 判断消息文本是否以指定前缀开头。
func (m *Message) IsCommand(prefix string) bool {
	return strings.HasPrefix(m.Text, prefix)
}

// SessionKey 返回该消息对应的会话键值。
func (m *Message) SessionKey() string {
	if m.IsDM {
		return "dm_" + m.UserID
	}
	return m.ChannelID
}

// ShouldReply 判断是否回复。私聊始终回复。
func ShouldReply(msg *Message, mode string) bool {
	if msg.IsDM {
		return true
	}
	switch mode {
	case "all":
		return true
	default:
		return msg.IsAtBot
	}
}
