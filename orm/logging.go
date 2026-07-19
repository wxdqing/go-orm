package orm

import "sync/atomic"

// Logger is the logging capability used by go-orm.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// LoggerFuncs adapts package-level logging functions to Logger.
type LoggerFuncs struct {
	DebugfFunc func(format string, args ...any)
	InfofFunc  func(format string, args ...any)
	WarnfFunc  func(format string, args ...any)
	ErrorfFunc func(format string, args ...any)
}

func (f LoggerFuncs) Debugf(format string, args ...any) {
	if f.DebugfFunc != nil {
		f.DebugfFunc(format, args...)
	}
}

func (f LoggerFuncs) Infof(format string, args ...any) {
	if f.InfofFunc != nil {
		f.InfofFunc(format, args...)
	}
}

func (f LoggerFuncs) Warnf(format string, args ...any) {
	if f.WarnfFunc != nil {
		f.WarnfFunc(format, args...)
	}
}

func (f LoggerFuncs) Errorf(format string, args ...any) {
	if f.ErrorfFunc != nil {
		f.ErrorfFunc(format, args...)
	}
}

var noopLogger Logger = LoggerFuncs{}

type loggerHolder struct {
	logger Logger
}

var processLogger atomic.Pointer[loggerHolder]

// SetLogger configures the process-wide go-orm logger. A nil logger restores no-op logging.
func SetLogger(logger Logger) {
	if logger == nil {
		logger = noopLogger
	}
	processLogger.Store(&loggerHolder{logger: logger})
}

// GetLogger returns the currently configured process-wide logger.
func GetLogger() Logger {
	if holder := processLogger.Load(); holder != nil {
		return holder.logger
	}
	return noopLogger
}
