/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/CESSProject/cess-miner/configs"
	"github.com/natefinch/lumberjack"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Log(level string, msg string)
	Pnc(msg string)
	Space(level string, msg string)
	Report(level string, msg string)
	Replace(level string, msg string)
	Ichal(level string, msg string)
	Schal(level string, msg string)
	Stag(level string, msg string)
	Restore(level string, msg string)
	Del(level string, msg string)
	Discover(level, msg string)
}

type Lg struct {
	logpath map[string]string
	log     map[string]*zap.Logger
}

var LogFiles = []string{
	"log",
	"panic",
	"space",
	"report",
	"replace",
	"ichal",
	"schal",
	"stag",
	"restore",
	"del",
	"discover",
}

var _ Logger = (*Lg)(nil)

func NewLogs(logfiles map[string]string) (*Lg, error) {
	var (
		logpath = make(map[string]string, 0)
		logCli  = make(map[string]*zap.Logger)
	)
	for name, fpath := range logfiles {
		dir := getFilePath(fpath)
		_, err := os.Stat(dir)
		if err != nil {
			err = os.MkdirAll(dir, configs.FileMode)
			if err != nil {
				return &Lg{}, errors.Errorf("%v,%v", dir, err)
			}
		}
		Encoder := getEncoder()
		newCore := zapcore.NewTee(
			zapcore.NewCore(Encoder, getWriteSyncer(fpath), zap.NewAtomicLevel()),
		)
		logpath[name] = fpath
		logCli[name] = zap.New(newCore, zap.AddCaller())
		logCli[name].Sugar().Infof("%v", fpath)
	}
	return &Lg{
		logpath: logpath,
		log:     logCli,
	}, nil
}

func (l *Lg) Log(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["log"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Pnc(msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["panic"]
	if ok {
		v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *Lg) Space(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["space"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Report(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["report"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Replace(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["replace"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Ichal(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["ichal"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Schal(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["schal"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Stag(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["stag"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Restore(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["restore"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Del(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["del"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func (l *Lg) Discover(level, msg string) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["discover"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
		case "err":
			v.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
		}
	}
}

func getFilePath(fpath string) string {
	path, _ := filepath.Abs(fpath)
	index := strings.LastIndex(path, string(os.PathSeparator))
	ret := path[:index]
	return ret
}

func getEncoder() zapcore.Encoder {
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
			EncodeCaller:   nil,
		})
}

func getWriteSyncer(fpath string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fpath,
		MaxSize:    10,
		MaxBackups: 10,
		MaxAge:     30,
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
