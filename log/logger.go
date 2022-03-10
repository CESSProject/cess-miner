package log

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"storage-mining/tools"
)

const (
	FlagDate      = 1 << iota // the date in local time zone
	FlagTime                  // the time in local time zone
	FlagLongFile              // full file name and line number: /a/b/c/d.go:23
	FlagShortFile             // final file name element and line number: d.go:23. overrides
	FlagUTC                   // if FlagDate or FlagTime is set, use UTC rather than the local time zone

	stdFlags = FlagDate | FlagTime // initial values for the standard logger
)

const (
	LvlError = iota
	LvlWarn
	LvlInfo
	LvlDebug
	LvlTrace
)

type Lvl int

// String returns a 5-character string containing the name of a Lvl.
func (l Lvl) String() string {
	switch l {
	case LvlTrace:
		return "TRACE"
	case LvlDebug:
		return "DEBUG"
	case LvlInfo:
		return "INFO "
	case LvlWarn:
		return "WARN "
	case LvlError:
		return "ERROR"
	default:
		panic("bad level")
	}
}

type Logger struct {
	sync.Mutex
	dst    *bufio.Writer
	prefix string
	flag   int
	level  Lvl
}

func New(w io.Writer, prefix string, flag int, level Lvl) *Logger {
	return &Logger{
		dst:    bufio.NewWriter(w),
		prefix: prefix,
		flag:   flag,
		level:  level,
	}
}

var std = New(os.Stderr, "", stdFlags, LvlDebug)

func (l *Logger) SetDst(w io.Writer) {
	l.Lock()
	defer l.Unlock()
	l.dst = bufio.NewWriter(w)
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(i int, wid int) []byte {
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// last one append
	b[bp] = byte('0' + i)
	return b[bp:]
}

// formatHeader fix header format
func (l *Logger) formatHeader(t time.Time, file string, line int) error {
	_, err := l.dst.Write(tools.S2B(l.prefix))
	if err != nil {
		return err
	}

	if l.flag&(FlagDate|FlagTime) != 0 {
		if l.flag&FlagUTC != 0 {
			t = t.UTC()
		}
		if l.flag&FlagDate != 0 {
			year, month, day := t.Date()
			_, err = l.dst.Write(itoa(year, 4))
			if err != nil {
				return err
			}
			_, err = l.dst.Write([]byte{'-'})
			if err != nil {
				return err
			}

			_, err = l.dst.Write(itoa(int(month), 2))
			if err != nil {
				return err
			}
			_, err = l.dst.Write([]byte{'-'})
			if err != nil {
				return err
			}

			_, err = l.dst.Write(itoa(day, 2))
			if err != nil {
				return err
			}
			_, err = l.dst.Write([]byte{' '})
			if err != nil {
				return err
			}
		}
		if l.flag&(FlagTime) != 0 {
			hour, min, sec := t.Clock()
			_, err = l.dst.Write(itoa(hour, 2))
			if err != nil {
				return err
			}
			_, err = l.dst.Write([]byte{':'})
			if err != nil {
				return err
			}

			_, err = l.dst.Write(itoa(min, 2))
			if err != nil {
				return err
			}
			_, err = l.dst.Write([]byte{':'})
			if err != nil {
				return err
			}

			_, err = l.dst.Write(itoa(sec, 2))
			if err != nil {
				return err
			}
			_, err = l.dst.Write([]byte{' '})
			if err != nil {
				return err
			}
		}
	}
	if l.flag&(FlagShortFile|FlagLongFile) != 0 {
		if l.flag&FlagShortFile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		_, err = l.dst.Write(tools.S2B(file))
		if err != nil {
			return err
		}
		_, err = l.dst.Write([]byte{':'})
		if err != nil {
			return err
		}

		_, err = l.dst.Write(itoa(line, -1))
		if err != nil {
			return err
		}
		_, err = l.dst.Write([]byte{':', ' '})
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Logger) Write(callDepth int, s string, lvl Lvl) error {
	// if the level granter the setting value then skip record the log
	if lvl > l.level {
		return nil
	}

	now := time.Now() // get this early.
	var file string
	var line int
	l.Lock()
	defer l.Unlock()
	if l.flag&(FlagShortFile|FlagLongFile) != 0 {
		// Release lock while getting caller info - it's expensive.
		l.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(callDepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.Lock()
	}

	err := l.formatHeader(now, file, line)
	if err != nil {
		return err
	}
	_, err = l.dst.Write(tools.S2B(s))
	if err != nil {
		return err
	}

	if len(s) == 0 || s[len(s)-1] != '\n' {
		_, err = l.dst.Write([]byte{'\n'})
		if err != nil {
			return err
		}
	}
	return nil
}

// Printf calls l.Write to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Write(2, fmt.Sprintf(format, v...), LvlInfo)
}

func (l *Logger) Print(v ...interface{}) {
	l.Write(2, fmt.Sprint(v...), LvlInfo)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.Write(2, fmt.Sprint(v...), LvlError)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Write(2, fmt.Sprintf(format, v...), LvlError)
	os.Exit(1)
}

func (l *Logger) Panic(v interface{}) {
	l.Write(2, fmt.Sprint(v), LvlError)
	panic(v)
}

func (l *Logger) Error(v ...interface{}) {
	l.Write(2, fmt.Sprint(v...), LvlError)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Write(2, fmt.Sprintf(format, v...), LvlError)
}

func (l *Logger) Info(v ...interface{}) {
	l.Print(v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.Printf(format, v...)
}

func (l *Logger) Debug(v ...interface{}) {
	l.Write(2, fmt.Sprint(v...), LvlDebug)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Write(2, fmt.Sprintf(format, v...), LvlDebug)
}

func (l *Logger) Trace(v ...interface{}) {
	l.Write(2, fmt.Sprint(v...), LvlTrace)
}

func (l *Logger) Tracef(format string, v ...interface{}) {
	l.Write(2, fmt.Sprintf(format, v...), LvlTrace)
}

func (l *Logger) Warn(v ...interface{}) {
	l.Write(2, fmt.Sprint(v...), LvlWarn)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Write(2, fmt.Sprintf(format, v...), LvlWarn)
}

func (l *Logger) SetPrefix(prefix string) {
	l.Lock()
	defer l.Unlock()
	l.prefix = prefix
}

func (l *Logger) SetFlag(flag int) {
	l.Lock()
	defer l.Unlock()
	l.flag = flag
}

func (l *Logger) Flush() {
	l.dst.Flush()
}

func SetFlag(flag int) {
	std.SetFlag(flag)
}

func SetPrefix(prefix string) {
	std.SetPrefix(prefix)
}

func SetDst(w io.Writer) {
	std.SetDst(w)
}

func Print(v ...interface{}) {
	std.Print(v...)
}

func Printf(format string, v ...interface{}) {
	std.Printf(format, v...)
}

func Fatal(v ...interface{}) {
	std.Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	std.Fatalf(format, v...)
}

func Info(v ...interface{}) {
	std.Info(v...)
}

func Infof(format string, v ...interface{}) {
	std.Infof(format, v...)
}

func Error(v ...interface{}) {
	std.Error(v...)
}

func Errorf(format string, v ...interface{}) {
	std.Errorf(format, v...)
}

func Debug(v ...interface{}) {
	std.Debug(v...)
}

func Debugf(format string, v ...interface{}) {
	std.Debugf(format, v...)
}

func Trace(v ...interface{}) {
	std.Trace(v...)
}

func Tracef(format string, v ...interface{}) {
	std.Tracef(format, v...)
}

func Warn(v ...interface{}) {
	std.Warn(v...)
}

func Warnf(format string, v ...interface{}) {
	std.Warnf(format, v...)
}

func Flush() {
	std.Flush()
}
