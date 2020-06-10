package sli2zap

import (
	"github.com/jealone/sli4go"
	"go.uber.org/zap"
)

func RegisterZap(logger *zap.Logger) {
	sli4go.InitLogger(&wrapperSugar{logger.Sugar()})
}

type wrapperSugar struct {
	*zap.SugaredLogger
}

func (l *wrapperSugar) Flush() error {
	return l.Sync()
}

func (l *wrapperSugar) Trace(v ...interface{}) {
	l.Debug(v...)
}

func (l *wrapperSugar) Tracef(format string, v ...interface{}) {
	l.Debugf(format, v...)
}

func (l *wrapperSugar) Traceln(i ...interface{}) {
	i = append(i, "\n")
	l.Info(i...)
}

func (l *wrapperSugar) Debugln(i ...interface{}) {
	i = append(i, "\n")
	l.Debug(i...)
}

func (l *wrapperSugar) Infoln(i ...interface{}) {
	i = append(i, "\n")
	l.Info(i...)
}

func (l *wrapperSugar) Warnln(i ...interface{}) {
	i = append(i, "\n")
	l.Warn(i...)
}

func (l *wrapperSugar) Errorln(i ...interface{}) {
	i = append(i, "\n")
	l.Error(i...)
}

func (l *wrapperSugar) Fatalln(i ...interface{}) {
	i = append(i, "\n")
	l.Fatal(i...)
}

func (l *wrapperSugar) Panicln(i ...interface{}) {
	i = append(i, "\n")
	l.Panic(i...)
}
