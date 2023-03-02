package logger

import (
	"log"
	"sync"
)

type Logger interface {
	Printf(format string, args ...interface{})
}

// NewLogger create an internal logger struct
// revive:disable:unexported-return
func NewLogger(prefix string, level LogLevel) (*defaultLogger, error) {
	return &defaultLogger{
		prefix:   prefix,
		logger:   log.Default(),
		logLevel: level,
	}, nil
}

//revive:enable:unexported-return

// Log is the internal logger struct
type Log interface {
	// Log a message with different LogLevel
	Log(lvl LogLevel, format string, v ...interface{})

	// SetLogger update the log system at run time
	SetLogger(l Logger, lvl LogLevel)

	// GetLogger return logger and level
	GetLogger() (Logger, LogLevel)
}

var _ Log = (*defaultLogger)(nil)

type defaultLogger struct {
	logger   Logger
	logLevel LogLevel
	prefix   string
	logGuard sync.RWMutex
}

func (l *defaultLogger) Log(lvl LogLevel, format string, v ...interface{}) {
	logger, logLvl := l.GetLogger()
	if logger == nil {
		return
	}

	if logLvl > lvl {
		return
	}
	logger.Printf(lvl.String()+" "+l.prefix+" "+format, v...)
}

func (l *defaultLogger) GetLogger() (Logger, LogLevel) {
	l.logGuard.RLock()
	defer l.logGuard.RUnlock()

	return l.logger, l.logLevel
}

func (l *defaultLogger) SetLogger(logger Logger, lvl LogLevel) {
	l.logGuard.Lock()
	defer l.logGuard.Unlock()

	l.logger = logger
	l.logLevel = lvl
}
