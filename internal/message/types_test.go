// internal/message/types_test.go
package message

import "testing"

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

func TestSessionKey(t *testing.T) {
	msgGroup := &Message{GroupID: "12345"}
	if key := msgGroup.SessionKey(); key != "group_12345" {
		t.Errorf("expected group_12345, got %s", key)
	}
}
