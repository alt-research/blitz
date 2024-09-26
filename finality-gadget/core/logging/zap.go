package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel string

const (
	Development LogLevel = "development" // prints debug and above
	Production  LogLevel = "production"  // prints info and above
)

type ZapLogger struct {
	logger *zap.Logger
}

var _ Logger = (*ZapLogger)(nil)

func NewLogLevel(isProduction bool) LogLevel {
	if isProduction {
		return Production
	} else {
		return Development
	}
}

// TODO: add a zap inner for logger interface.
func NewZapLogger(env LogLevel) (*ZapLogger, error) {
	config := zap.NewProductionConfig()
	if env == Development {
		config = zap.NewDevelopmentConfig()
	}

	config.DisableStacktrace = true
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if env == Development {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
	return &ZapLogger{
		logger: logger,
	}, nil
}

func (z *ZapLogger) Inner() *zap.Logger {
	return z.logger
}

func (z *ZapLogger) Debug(msg string, tags ...any) {
	z.logger.Sugar().Debugw(msg, tags...)
}

func (z *ZapLogger) Info(msg string, tags ...any) {
	z.logger.Sugar().Infow(msg, tags...)
}

func (z *ZapLogger) Warn(msg string, tags ...any) {
	z.logger.Sugar().Warnw(msg, tags...)
}

func (z *ZapLogger) Error(msg string, tags ...any) {
	z.logger.Sugar().Errorw(msg, tags...)
}

func (z *ZapLogger) Fatal(msg string, tags ...any) {
	z.logger.Sugar().Fatalw(msg, tags...)
}

func (z *ZapLogger) Debugf(template string, args ...interface{}) {
	z.logger.Sugar().Debugf(template, args...)
}

func (z *ZapLogger) Infof(template string, args ...interface{}) {
	z.logger.Sugar().Infof(template, args...)
}

func (z *ZapLogger) Warnf(template string, args ...interface{}) {
	z.logger.Sugar().Warnf(template, args...)
}

func (z *ZapLogger) Errorf(template string, args ...interface{}) {
	z.logger.Sugar().Errorf(template, args...)
}

func (z *ZapLogger) Fatalf(template string, args ...interface{}) {
	z.logger.Sugar().Fatalf(template, args...)
}

func (z *ZapLogger) With(tags ...any) Logger {
	return &ZapLogger{
		logger: z.logger.Sugar().With(tags...).Desugar(),
	}
}
