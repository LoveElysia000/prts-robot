// cmd/bot/main.go
package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/loveelysia000/robot/internal/core"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	// 同时输出到控制台（Docker logs）和文件
	setupLogging()

	slog.Info("loading config", "path", cfgPath)
	bot, err := core.NewBot(cfgPath)
	if err != nil {
		slog.Error("failed to init bot", "err", err)
		os.Exit(1)
	}

	bot.Run()
}

func setupLogging() {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return
	}
	file, err := os.OpenFile("logs/bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	handler := slog.NewJSONHandler(io.MultiWriter(os.Stdout, file), &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}
