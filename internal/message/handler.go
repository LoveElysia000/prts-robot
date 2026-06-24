// Package message 定义消息结构和消息处理逻辑，包括消息类型、触发判断和命令解析。
package message

// TriggerConfig 表示消息触发配置，包含触发模式和命令前缀。
type TriggerConfig struct {
	Mode          string
	CommandPrefix string
}

// ShouldReply 根据触发配置判断是否需要回复该消息。私聊始终回复。
func ShouldReply(msg *Message, cfg TriggerConfig) bool {
	if msg.IsPrivate() {
		return true
	}
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
