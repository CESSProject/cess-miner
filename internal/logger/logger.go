package logger

import (
	"cess-bucket/configs"
	"fmt"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Warn *zap.Logger
	Err  *zap.Logger
	Out  *zap.Logger
)

func LoggerInit() {
	_, err := os.Stat(configs.MinerDataPath + configs.LogfilePathPrefix)
	if err != nil {
		err = os.MkdirAll(configs.MinerDataPath+configs.LogfilePathPrefix, os.ModeDir)
		if err != nil {
			configs.LogfilePathPrefix = "./log"
		} else {
			configs.LogfilePathPrefix = configs.MinerDataPath + configs.LogfilePathPrefix
		}
	} else {
		configs.LogfilePathPrefix = configs.MinerDataPath + configs.LogfilePathPrefix
	}
	initOutLogger()
	initWarnLogger()
	initErrLogger()
}

// out log
func initOutLogger() {
	outlogpath := configs.LogfilePathPrefix + "/out.log"
	hook := lumberjack.Logger{
		Filename:   outlogpath,
		MaxSize:    50,  //MB
		MaxAge:     365, //Day
		MaxBackups: 0,
		LocalTime:  true,
		Compress:   true,
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		CallerKey:    "file",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   formatEncodeTime,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.InfoLevel)
	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...),
		atomicLevel,
	)
	caller := zap.AddCaller()
	development := zap.Development()
	Out = zap.New(core, caller, development)
	Out.Sugar().Errorf("The service has started and created a log file in the %v", outlogpath)
}

// warn log
func initWarnLogger() {
	warnlogpath := configs.LogfilePathPrefix + "/warn.log"
	hook := lumberjack.Logger{
		Filename:   warnlogpath,
		MaxSize:    10,  //MB
		MaxAge:     365, //Day
		MaxBackups: 0,
		LocalTime:  true,
		Compress:   true,
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		CallerKey:    "file",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   formatEncodeTime,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.WarnLevel)
	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...),
		atomicLevel,
	)
	caller := zap.AddCaller()
	development := zap.Development()
	Warn = zap.New(core, caller, development)
	Warn.Sugar().Warnf("The service has started and created a log file in the %v", warnlogpath)
}

// error log
func initErrLogger() {
	errlogpath := configs.LogfilePathPrefix + "/error.log"
	hook := lumberjack.Logger{
		Filename:   errlogpath,
		MaxSize:    10,  //MB
		MaxAge:     365, //Day
		MaxBackups: 0,
		LocalTime:  true,
		Compress:   true,
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		CallerKey:    "file",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   formatEncodeTime,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.ErrorLevel)
	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...),
		atomicLevel,
	)
	caller := zap.AddCaller()
	development := zap.Development()
	Err = zap.New(core, caller, development)
	Err.Sugar().Errorf("The service has started and created a log file in the %v", errlogpath)
}

func formatEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()))
}
