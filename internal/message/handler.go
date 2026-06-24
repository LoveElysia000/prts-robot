// internal/message/handler.go
package message

type TriggerConfig struct {
	Mode          string
	CommandPrefix string
}

func ShouldReply(msg *Message, cfg TriggerConfig) bool {
	switch cfg.Mode {
	case "all":
		return true
	case "at":
		return msg.IsAtBot
	case "hybrid":
		return msg.IsAtBot
	}
	return false
}
