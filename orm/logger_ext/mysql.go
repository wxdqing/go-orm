package logger_ext

import (
	logger "gitee.com/wxdqing/logger.git"
	dblog "gorm.io/gorm/logger"
	"log"
	"time"
)

func NewDbLogger(params ...any) dblog.Interface {
	logLevel := dblog.Info
	if len(params) > 0 {
		logLevel = params[0].(dblog.LogLevel)
	}
	return dblog.New(log.New(logger.Writer, "\r\n", log.LstdFlags), dblog.Config{
		SlowThreshold:             200 * time.Millisecond,
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
		LogLevel:                  logLevel,
	})
}
