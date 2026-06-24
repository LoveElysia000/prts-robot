// internal/message/handler_test.go
package message

import "testing"

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
