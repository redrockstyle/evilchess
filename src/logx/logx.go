package logx

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	InitLogger(w io.Writer)
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	DPanic(args ...interface{})
	DPanicf(template string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
}

type Logx struct {
	level       zapcore.Level
	dev         bool
	console     bool
	sugarLogger *zap.SugaredLogger
}

func NewLogx(lvl zapcore.Level, dev bool, console bool) *Logx {
	return &Logx{level: lvl, dev: dev, console: console}
}

var loggerLevelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func GetLoggerLevelByString(lvl string) zapcore.Level {
	level, exist := loggerLevelMap[lvl]
	if !exist {
		return zapcore.DebugLevel
	}

	return level
}

func (l *Logx) InitLogger(w io.Writer) {
	var logWriter zapcore.WriteSyncer
	if l.console {
		logWriter = zapcore.AddSync(os.Stdout)
	} else {
		logWriter = zapcore.AddSync(w)
	}

	var encoderCfg zapcore.EncoderConfig
	if l.dev {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	var encoder zapcore.Encoder
	encoderCfg.LevelKey = "LEVEL"
	encoderCfg.CallerKey = "CALLER"
	encoderCfg.TimeKey = "TIME"
	encoderCfg.NameKey = "NAME"
	encoderCfg.MessageKey = "MESSAGE"

	if l.console {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(encoder, logWriter, zap.NewAtomicLevelAt(l.level))
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	l.sugarLogger = logger.Sugar()
	if err := l.sugarLogger.Sync(); err != nil {
		l.sugarLogger.Error(err)
	}
}

func (l *Logx) Debug(args ...interface{}) {
	l.sugarLogger.Debug(args...)
}

func (l *Logx) Debugf(template string, args ...interface{}) {
	l.sugarLogger.Debugf(template, args...)
}

func (l *Logx) Info(args ...interface{}) {
	l.sugarLogger.Info(args...)
}

func (l *Logx) Infof(template string, args ...interface{}) {
	l.sugarLogger.Infof(template, args...)
}

func (l *Logx) Warn(args ...interface{}) {
	l.sugarLogger.Warn(args...)
}

func (l *Logx) Warnf(template string, args ...interface{}) {
	l.sugarLogger.Warnf(template, args...)
}

func (l *Logx) Error(args ...interface{}) {
	l.sugarLogger.Error(args...)
}

func (l *Logx) Errorf(template string, args ...interface{}) {
	l.sugarLogger.Errorf(template, args...)
}

func (l *Logx) DPanic(args ...interface{}) {
	l.sugarLogger.DPanic(args...)
}

func (l *Logx) DPanicf(template string, args ...interface{}) {
	l.sugarLogger.DPanicf(template, args...)
}

func (l *Logx) Panic(args ...interface{}) {
	l.sugarLogger.Panic(args...)
}

func (l *Logx) Panicf(template string, args ...interface{}) {
	l.sugarLogger.Panicf(template, args...)
}

func (l *Logx) Fatal(args ...interface{}) {
	l.sugarLogger.Fatal(args...)
}

func (l *Logx) Fatalf(template string, args ...interface{}) {
	l.sugarLogger.Fatalf(template, args...)
}
