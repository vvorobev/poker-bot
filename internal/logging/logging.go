package logging

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Setup configures the global slog logger. If logPath is empty, logs go to stdout.
// Rotation: 10 MB per file, 5 backups kept.
func Setup(logPath string) {
	var w io.Writer
	if logPath == "" {
		w = os.Stdout
	} else {
		w = &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    10, // MB
			MaxBackups: 5,
		}
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(h))
}
