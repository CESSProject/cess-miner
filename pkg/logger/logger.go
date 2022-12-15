/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package logger

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ILog interface {
	Log(string, string, error)
	Pnc(string, error)
	Common(string, error)
	Upfile(string, error)
}

type logs struct {
	logpath map[string]string
	log     map[string]*zap.Logger
}

func NewLogs(logfiles map[string]string) (ILog, error) {
	var (
		logpath = make(map[string]string, 0)
		logCli  = make(map[string]*zap.Logger)
	)
	for name, fpath := range logfiles {
		dir := getFilePath(fpath)
		_, err := os.Stat(dir)
		if err != nil {
			err = os.MkdirAll(dir, os.ModeDir)
			if err != nil {
				return nil, errors.Errorf("%v,%v", dir, err)
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
	return &logs{
		logpath: logpath,
		log:     logCli,
	}, nil
}

func (l *logs) Log(name, level string, err error) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log[name]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %v", filepath.Base(file), line, err)
		case "error", "err":
			v.Sugar().Errorf("[%v:%d] %v", filepath.Base(file), line, err)
		case "warn":
			v.Sugar().Warnf("[%v:%d] %v", filepath.Base(file), line, err)
		}
	}
}

func (l *logs) Pnc(level string, err error) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["panic"]
	if ok {
		switch level {
		case "error", "err":
			v.Sugar().Errorf("[%v:%d] %v", filepath.Base(file), line, err)
		}
	}
}

func (l *logs) Common(level string, err error) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["common"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %v", filepath.Base(file), line, err)
		case "error", "err":
			v.Sugar().Errorf("[%v:%d] %v", filepath.Base(file), line, err)
		case "warn":
			v.Sugar().Warnf("[%v:%d] %v", filepath.Base(file), line, err)
		}
	}
}

func (l *logs) Upfile(level string, err error) {
	_, file, line, _ := runtime.Caller(1)
	v, ok := l.log["upfile"]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("[%v:%d] %v", filepath.Base(file), line, err)
		case "error", "err":
			v.Sugar().Errorf("[%v:%d] %v", filepath.Base(file), line, err)
		case "warn":
			v.Sugar().Warnf("[%v:%d] %v", filepath.Base(file), line, err)
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
