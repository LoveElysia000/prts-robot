// cmd/bot/main.go
//
// Package main 是机器人的入口点，负责加载配置并启动 QQ 机器人服务。
package main

import (
	"log/slog"
	"os"

	"github.com/loveelysia000/robot/internal/core"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	slog.Info("loading config", "path", cfgPath)
	bot, err := core.NewBot(cfgPath)
	if err != nil {
		slog.Error("failed to init bot", "err", err)
		os.Exit(1)
	}

	if err := bot.Run(); err != nil {
		slog.Error("bot stopped", "err", err)
		os.Exit(1)
	}
}
