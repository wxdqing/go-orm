package sql

import (
	"context"
	"fmt"

	"github.com/wxdqing/go-orm/orm/driverapi"
	logger "git.wxdqing.com/sprout/logger.git"
)

// MySQL 基于 GORM 的 MySQL 实现。
type MySQL struct {
	gormBase
}

func NewMySQL() driverapi.Driver {
	return &MySQL{}
}

func (m *MySQL) InitDB(ctx context.Context, o *driverapi.Options) error {
	DB, err := openMysqlDB(o.Conf.Mysql)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	m.DB = DB
	if err := m.finishInit(o, driverapi.TypeMySQL); err != nil {
		return err
	}
	logger.Infof("current use db:  %v shard_mode=%s", o.Conf.Mysql, o.Conf.Mysql.Shard.Mode)
	return nil
}
