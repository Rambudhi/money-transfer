package log

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func Init() {
	Logger = slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		),
	)
}

func getLogger() *slog.Logger {
	if Logger == nil {
		return slog.Default()
	}

	return Logger
}

func Info(message string, args ...any) {
	getLogger().Info(message, args...)
}

func Error(message string, args ...any) {
	getLogger().Error(message, args...)
}

func Warn(message string, args ...any) {
	getLogger().Warn(message, args...)
}

func Debug(message string, args ...any) {
	getLogger().Debug(message, args...)
}
