package logger

import (
	"cess-bucket/configs"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Out *zap.Logger
	Err *zap.Logger
	Uld *zap.Logger
	Dld *zap.Logger
	Flr *zap.Logger
	Pnc *zap.Logger
	Chg *zap.Logger
)

func LoggerInit() {
	_, err := os.Stat(configs.LogfileDir)
	if err != nil {
		err = os.MkdirAll(configs.LogfileDir, os.ModeDir)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(1)
		}
	}

	var log_file = []string{
		"common.log",
		"error.log",
		"upfile.log",
		"downfile.log",
		"filler.log",
		"panic.log",
		"challenge.log",
	}

	for i := 0; i < len(log_file); i++ {
		Encoder := GetEncoder()
		fpath := filepath.Join(configs.LogfileDir, log_file[i])
		WriteSyncer := GetWriteSyncer(fpath)
		newCore := zapcore.NewTee(zapcore.NewCore(Encoder, WriteSyncer, zap.NewAtomicLevel()))
		switch i {
		case 0:
			Out = zap.New(newCore, zap.AddCaller())
			Out.Sugar().Infof("%v", fpath)
		case 1:
			Err = zap.New(newCore, zap.AddCaller())
			Err.Sugar().Infof("%v", fpath)
		case 2:
			Uld = zap.New(newCore, zap.AddCaller())
			Uld.Sugar().Infof("%v", fpath)
		case 3:
			Dld = zap.New(newCore, zap.AddCaller())
			Dld.Sugar().Infof("%v", fpath)
		case 4:
			Flr = zap.New(newCore, zap.AddCaller())
			Flr.Sugar().Infof("%v", fpath)
		case 5:
			Pnc = zap.New(newCore, zap.AddCaller())
			Pnc.Sugar().Infof("%v", fpath)
		case 6:
			Chg = zap.New(newCore, zap.AddCaller())
			Chg.Sugar().Infof("%v", fpath)
		}
	}
	log.Println("log loaded successfully")
}

func GetEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(
		zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller_line",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    cEncodeLevel,
			EncodeTime:     cEncodeTime,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   cEncodeCaller,
		})
}

func GetWriteSyncer(fpath string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fpath,
		MaxSize:    30,
		MaxBackups: 99,
		MaxAge:     180,
		LocalTime:  true,
		Compress:   true,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func cEncodeLevel(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func cEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format("2006-01-02 15:04:05") + "]")
}

func cEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + caller.TrimmedPath() + "]")
}
