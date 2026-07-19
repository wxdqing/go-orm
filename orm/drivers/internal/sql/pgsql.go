package sql

import (
	"context"
	"fmt"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
)

// Pgsql 基于 GORM 的 PostgreSQL 实现。
type Pgsql struct {
	gormBase
}

func NewPgsql() driverapi.CoreDriver {
	return &Pgsql{}
}

var _ driverapi.SQLDriver = (*Pgsql)(nil)

func (p *Pgsql) InitDB(ctx context.Context, o *driverapi.Options) error {
	DB, err := openPgsqlDB(o.Conf.Pgsql)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	p.DB = DB
	if err := p.finishInit(o, driverapi.TypePostgresSQL); err != nil {
		return err
	}
	orm.GetLogger().Infof("current use db: postgres shard_mode=%s", o.Conf.Pgsql.Shard.Mode)
	return nil
}
