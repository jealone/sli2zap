package sli2zap

import (
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jealone/sli4go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	Debug = zapcore.DebugLevel
	Info  = zapcore.InfoLevel
	Warn  = zapcore.WarnLevel
	Error = zapcore.ErrorLevel
	Panic = zapcore.PanicLevel
	Fatal = zapcore.FatalLevel
)

const (
	SIGROTATE = syscall.SIGHUP
)

type (
	Logger        = zap.Logger
	Level         = zapcore.Level
	EncoderConfig = zapcore.EncoderConfig
	Option        = zap.Option
)

type LogConfig struct {
	Logfile       string         `yaml:"logfile"`
	MaxSize       int            `yaml:"max_size"`
	MaxBackups    int            `yaml:"max_backups"`
	MaxAge        int            `yaml:"max_age"`
	Compress      bool           `yaml:"compress"`
	Level         string         `yaml:"level"`
	Trace         bool           `yaml:"trace"`
	TraceSkip     int            `yaml:"trace_skip"`
	EncoderConfig *EncoderConfig `yaml:"encoder"`
}

func (c *LogConfig) GetMaxSize() int {
	return c.MaxSize
}

func (c *LogConfig) GetMaxAge() int {
	return c.MaxAge
}

func (c *LogConfig) GetMaxBackups() int {
	return c.MaxBackups
}

func (c *LogConfig) GetTrace() bool {
	return c.Trace
}

func (c *LogConfig) GetTraceSkip() int {
	return c.TraceSkip
}

func (c *LogConfig) GetCompress() bool {
	return c.Compress
}

func (c *LogConfig) GetLogfile() string {
	if "" == c.Logfile {
		return "logs/error.log"

	}
	return c.Logfile
}

func NewProductionEncoderConfig() EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func (c *LogConfig) GetEncoderConfig() EncoderConfig {
	if nil == c.EncoderConfig {
		return NewProductionEncoderConfig()
	}

	enc := c.EncoderConfig
	if "" == enc.LineEnding {
		enc.LineEnding = zapcore.DefaultLineEnding
	}

	if nil == enc.EncodeLevel {
		enc.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	if nil == enc.EncodeTime {
		enc.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	if nil == enc.EncodeDuration {
		enc.EncodeDuration = zapcore.SecondsDurationEncoder
	}

	if nil == enc.EncodeCaller {
		enc.EncodeCaller = zapcore.ShortCallerEncoder
	}

	//if nil == enc.EncodeName {
	//	enc.EncodeName = zapcore.FullNameEncoder
	//}

	return *enc
}

func (c *LogConfig) GetLevel() Level {
	level := strings.ToUpper(c.Level)
	switch level {
	case "":
		return Info
	case "DEBUG":
		return Debug
	case "INFO":
		return Info
	case "WARN":
		return Warn
	case "ERROR":
		return Error
	case "PANIC":
		return Panic
	case "FATAL":
		return Fatal
	default:
		sli4go.Fatalf("invalid log level(%s)", c.Level)
		return 0
	}
}

func logFileCheck(path string) {
	abs, err := filepath.Abs(path)
	if nil != err {
		sli4go.Fatalf("get abs for file(%s) error: %s", path, err)
	}

	info, err := os.Stat(abs)

	if nil != err {
		if os.IsNotExist(err) {
			dir := filepath.Dir(abs)
			err = os.MkdirAll(dir, os.ModePerm)

			if nil != err {
				sli4go.Fatalf("mkdir %s error: %s", dir, err)
			}
		} else {
			sli4go.Fatalf("log file stat error: %s", err)
		}
	} else {
		if info.IsDir() {
			sli4go.Fatalf("the specific log file (%s) is directory", abs)
		}
	}

}

func coroutine(f func()) {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				sli4go.Errorf("recover_panic:%s\n", e)
			}
		}()
		f()
	}()
}

var (
	getSignalBroadcast = createSignalBroadcast()

	initDaemon = createDaemon(func() {

		c := make(chan os.Signal, 1)
		signal.Notify(c, SIGROTATE)

		coroutine(func() {
			for {
				select {
				case <-c:
					//getSignalBroadcast().L.Lock()
					getSignalBroadcast().Broadcast()
					//getSignalBroadcast().L.Unlock()
				}
			}
		})
	})
)

func createDaemon(f func()) func() {
	var (
		once sync.Once
	)
	return func() {
		once.Do(func() {
			f()
		})
	}
}

func createSignalBroadcast() func() *sync.Cond {
	var (
		cond *sync.Cond
		once sync.Once
	)

	return func() *sync.Cond {
		once.Do(func() {
			locker := &sync.Mutex{}
			cond = sync.NewCond(locker)
		})
		return cond
	}
}

func Broadcast() {
	//getSignalBroadcast().L.Lock()
	getSignalBroadcast().Broadcast()
	//getSignalBroadcast().L.Unlock()
}

type Notify interface {
	Broadcast()
}

type Decoder interface {
	Decode(v interface{}) (err error)
}

func DecodeLogger(dec Decoder, options ...Option) (error, *Logger) {
	conf := &LogConfig{}
	err := dec.Decode(conf)

	if nil != err {
		return err, nil
	}

	return nil, NewLogger(conf, options...)
}

func NewLogger(config *LogConfig, options ...Option) *Logger {

	initDaemon()

	c := make(chan os.Signal, 1)

	coroutine(func() {
		for {
			getSignalBroadcast().L.Lock()
			getSignalBroadcast().Wait()
			c <- SIGROTATE
			getSignalBroadcast().L.Unlock()
		}
	})

	rotater := newFileNotify(config, c)

	coroutine(func() {
		for {
			select {
			case <-c:
				_ = rotater.Rotate()
			}
		}
	})

	return newLogger(rotater, config, options...)
}

func newFileNotify(config *LogConfig, signals chan os.Signal) *lumberjack.Logger {

	logfile := config.GetLogfile()
	logFileCheck(logfile)

	coroutine(func() {
		ticker := time.Tick(time.Second * 10)
		for {
			select {
			case <-ticker:
				_, err := os.Stat(logfile)
				if nil != err && !os.IsExist(err) {
					signals <- SIGROTATE
				}
			}
		}
	})

	lumberJackLogger := &lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    config.GetMaxSize(),
		MaxBackups: config.GetMaxBackups(),
		MaxAge:     config.GetMaxAge(),
		Compress:   config.GetCompress(),
	}
	return lumberJackLogger
}

func newLogger(writer io.Writer, config *LogConfig, options ...Option) *Logger {
	encoder := zapcore.NewConsoleEncoder(config.GetEncoderConfig())
	writerSync := zapcore.AddSync(writer)
	core := zapcore.NewCore(encoder, writerSync, config.GetLevel())

	if config.GetTrace() {
		options = append(options, zap.AddCaller(), zap.AddCallerSkip(config.GetTraceSkip()))
	}

	return zap.New(core, options...)
}
