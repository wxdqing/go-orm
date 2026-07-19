package logger_ext

import (
	"log"
	"strings"
	"time"

	"github.com/wxdqing/go-orm/orm"
	dblog "gorm.io/gorm/logger"
)

type logWriter struct{}

func (logWriter) Write(data []byte) (int, error) {
	if message := strings.TrimSpace(string(data)); message != "" {
		orm.GetLogger().Infof("%s", message)
	}
	return len(data), nil
}

func NewDbLogger(params ...any) dblog.Interface {
	logLevel := dblog.Info
	if len(params) > 0 {
		logLevel = params[0].(dblog.LogLevel)
	}
	return dblog.New(log.New(logWriter{}, "", log.LstdFlags), dblog.Config{
		SlowThreshold:             200 * time.Millisecond,
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		LogLevel:                  logLevel,
	})
}
