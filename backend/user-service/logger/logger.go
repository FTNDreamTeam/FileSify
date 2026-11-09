package logger

import (
	"log/slog"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Init(logPath string) {
	rotator := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
		LocalTime:  true,
	}

	w := rotator

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     slog.LevelInfo, //Devug | -> Info, Warn, Error
		AddSource: false,
	})
	slog.SetDefault(slog.New(handler))
}
