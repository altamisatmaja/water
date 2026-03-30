package logger

import (
	"log/slog"
	"os"
)

var log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func SetLevel(level slog.Level) {
	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	log.Error(msg, args...)
}
