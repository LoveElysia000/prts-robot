// cmd/bot/main.go — Discord bot 入口，加载配置、初始化日志并启动。
package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/loveelysia000/robot/internal/core"
)

func main() {
	setupLogging()

	bot, err := core.NewBot("config.yaml")
	if err != nil {
		slog.Error("init bot failed", "err", err)
		os.Exit(1)
	}

	if err := bot.Run(); err != nil {
		slog.Error("bot stopped", "err", err)
		os.Exit(1)
	}
}

// setupLogging 将日志同时写入 stdout 和 logs/bot.log 文件。
func setupLogging() {
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
		slog.Warn("cannot open log file, logging to stdout only", "err", err)
		return
	}
	w := io.MultiWriter(os.Stdout, file)
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})))
}
