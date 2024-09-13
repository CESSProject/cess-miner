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
	// panic log
	Pnc(msg string)
	// level: info | err
	Log(level string, msg string)
	// level: info | err
	Space(level string, msg string)
	// level: info | err
	Report(level string, msg string)
	// level: info | err
	Replace(level string, msg string)
	// level: info | err
	Ichal(level string, msg string)
	// level: info | err
	Schal(level string, msg string)
	// level: info | err
	Stag(level string, msg string)
	// level: info | err
	Restore(level string, msg string)
	// level: info | err
	Del(level string, msg string)
	// level: info | err
	Putf(level, msg string)
	// level: info | err
	Getf(level, msg string)
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
	"delete",
	"putf",
	"getf",
}

type logs struct {
	logpath     map[string]string
	log_log     *zap.Logger
	log_pnc     *zap.Logger
	log_space   *zap.Logger
	log_report  *zap.Logger
	log_replace *zap.Logger
	log_ichal   *zap.Logger
	log_schal   *zap.Logger
	log_stag    *zap.Logger
	log_restore *zap.Logger
	log_del     *zap.Logger
	log_putf    *zap.Logger
	log_getf    *zap.Logger
}

var _ Logger = (*logs)(nil)

func NewLogs(logfiles map[string]string) (Logger, error) {
	var (
		l       = &logs{}
		logpath = make(map[string]string, 0)
	)
	for name, fpath := range logfiles {
		dir := getFilePath(fpath)
		_, err := os.Stat(dir)
		if err != nil {
			err = os.MkdirAll(dir, configs.FileMode)
			if err != nil {
				return nil, errors.Errorf("%v,%v", dir, err)
			}
		}
		Encoder := getEncoder()
		newCore := zapcore.NewTee(
			zapcore.NewCore(Encoder, getWriteSyncer(fpath), zap.NewAtomicLevel()),
		)
		logpath[name] = fpath
		switch name {
		case "log":
			l.log_log = zap.New(newCore, zap.AddCaller())
			l.log_log.Sugar().Infof("%v", fpath)
		case "panic":
			l.log_pnc = zap.New(newCore, zap.AddCaller())
			l.log_pnc.Sugar().Infof("%v", fpath)
		case "space":
			l.log_space = zap.New(newCore, zap.AddCaller())
			l.log_space.Sugar().Infof("%v", fpath)
		case "report":
			l.log_report = zap.New(newCore, zap.AddCaller())
			l.log_report.Sugar().Infof("%v", fpath)
		case "replace":
			l.log_replace = zap.New(newCore, zap.AddCaller())
			l.log_replace.Sugar().Infof("%v", fpath)
		case "ichal":
			l.log_ichal = zap.New(newCore, zap.AddCaller())
			l.log_ichal.Sugar().Infof("%v", fpath)
		case "schal":
			l.log_schal = zap.New(newCore, zap.AddCaller())
			l.log_schal.Sugar().Infof("%v", fpath)
		case "stag":
			l.log_stag = zap.New(newCore, zap.AddCaller())
			l.log_stag.Sugar().Infof("%v", fpath)
		case "restore":
			l.log_restore = zap.New(newCore, zap.AddCaller())
			l.log_restore.Sugar().Infof("%v", fpath)
		case "delete":
			l.log_del = zap.New(newCore, zap.AddCaller())
			l.log_del.Sugar().Infof("%v", fpath)
		case "putf":
			l.log_putf = zap.New(newCore, zap.AddCaller())
			l.log_putf.Sugar().Infof("%v", fpath)
		case "getf":
			l.log_getf = zap.New(newCore, zap.AddCaller())
			l.log_getf.Sugar().Infof("%v", fpath)
		}
	}
	l.logpath = logpath
	return l, nil
}

func (l *logs) Log(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_log.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_log.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Pnc(msg string) {
	_, file, line, _ := runtime.Caller(1)
	l.log_pnc.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
}

func (l *logs) Space(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_space.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_space.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Report(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_report.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_report.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Replace(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_replace.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_replace.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Ichal(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_ichal.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_ichal.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Schal(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_schal.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_schal.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Stag(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_stag.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_stag.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Restore(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_restore.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_restore.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Del(level string, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_del.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_del.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Putf(level, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_putf.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_putf.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
	}
}

func (l *logs) Getf(level, msg string) {
	_, file, line, _ := runtime.Caller(1)
	switch level {
	case "info":
		l.log_getf.Sugar().Infof("[%v:%d] %s", filepath.Base(file), line, msg)
	case "err":
		l.log_getf.Sugar().Errorf("[%v:%d] %s", filepath.Base(file), line, msg)
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
