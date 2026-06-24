// cmd/bot/main.go
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

	bot.Run()
}
