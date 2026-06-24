// Package message 定义消息结构和消息处理逻辑，包括消息类型、触发判断和命令解析。
package message

import "testing"

// TestShouldReply 验证 ShouldReply 在不同触发模式下能正确判断是否需要回复消息。
func TestShouldReply(t *testing.T) {
	cfg := TriggerConfig{Mode: "hybrid", CommandPrefix: "/"}
	if !ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: true}, cfg) {
		t.Error("hybrid: at msg should reply")
	}
	if ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfg) {
		t.Error("hybrid: non-at should not reply")
	}

	cfgAll := TriggerConfig{Mode: "all", CommandPrefix: "/"}
	if !ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfgAll) {
		t.Error("all: should reply")
	}

	cfgAt := TriggerConfig{Mode: "at", CommandPrefix: "/"}
	if ShouldReply(&Message{GroupID: "123", Text: "你好", IsAtBot: false}, cfgAt) {
		t.Error("at: non-at should not reply")
	}
}
