package cmd

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func newZapLogger(debug bool) *zap.Logger {
	zap.NewProductionConfig()
	encoderConfig := zapcore.EncoderConfig{
		LevelKey:       "level",
		MessageKey:     "msg",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "file",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(os.Stdout),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			if debug {
				return true
			}
			return lvl > zapcore.DebugLevel
		}))
	return zap.New(core)
}
