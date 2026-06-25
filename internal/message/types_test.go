package message

import "testing"

func TestShouldReply(t *testing.T) {
	// DM always replies
	if !ShouldReply(&Message{IsDM: true}, "mention") {
		t.Error("DM should reply")
	}

	// mention mode: only @
	if !ShouldReply(&Message{IsAtBot: true}, "mention") {
		t.Error("mention mode: at should reply")
	}
	if ShouldReply(&Message{IsAtBot: false}, "mention") {
		t.Error("mention mode: non-at should not reply")
	}

	// all mode
	if !ShouldReply(&Message{IsAtBot: false}, "all") {
		t.Error("all mode should reply")
	}
}

func TestIsCommand(t *testing.T) {
	msg := &Message{Text: "/help"}
	if !msg.IsCommand("/") {
		t.Error("/help is command")
	}
	msg2 := &Message{Text: "hello"}
	if msg2.IsCommand("/") {
		t.Error("hello is not command")
	}
}

func TestSessionKey(t *testing.T) {
	if k := (&Message{ChannelID: "ch1"}).SessionKey(); k != "ch1" {
		t.Errorf("expected ch1, got %s", k)
	}
	if k := (&Message{IsDM: true, UserID: "u1"}).SessionKey(); k != "dm_u1" {
		t.Errorf("expected dm_u1, got %s", k)
	}
}
