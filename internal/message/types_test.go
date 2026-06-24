// Package message 定义消息结构和消息处理逻辑，包括消息类型、触发判断和命令解析。
package message

import "testing"

// TestIsCommand 验证 IsCommand 能正确判断消息文本是否为命令。
func TestIsCommand(t *testing.T) {
	msg := &Message{Text: "/help"}
	if !msg.IsCommand("/") {
		t.Error("expected /help to be command")
	}
	msg2 := &Message{Text: "你好"}
	if msg2.IsCommand("/") {
		t.Error("expected 你好 not to be command")
	}
}

// TestSessionKey 验证 SessionKey 能正确生成基于群 ID 的会话键值。
func TestSessionKey(t *testing.T) {
	msgGroup := &Message{GroupID: "12345"}
	if key := msgGroup.SessionKey(); key != "group_12345" {
		t.Errorf("expected group_12345, got %s", key)
	}
}
