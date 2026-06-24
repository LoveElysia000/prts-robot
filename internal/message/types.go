// Package message 定义消息结构和消息处理逻辑，包括消息类型、触发判断和命令解析。
package message

import "strings"

// Message 表示一条来自 QQ 群的聊天消息，包含群 ID、用户 ID、文本内容、@ 状态和消息 ID。
type Message struct {
	GroupID string
	UserID  string
	Text    string
	IsAtBot bool
	MsgID   string
}

// IsCommand 判断消息文本是否以指定前缀开头，用于识别命令消息。
func (m *Message) IsCommand(prefix string) bool {
	return strings.HasPrefix(m.Text, prefix)
}

// SessionKey 返回该消息对应的会话键值，用于会话管理按群分组。
func (m *Message) SessionKey() string {
	return "group_" + m.GroupID
}
