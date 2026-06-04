package logic

import (
	"context"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest/metadata"
)

const shardTestCount = 4

func playerTableShardConf() orm.SQLShardConf {
	return orm.SQLShardConf{
		Mode:     orm.ShardModeTable,
		KeyField: "id",
		Tables: []orm.TableShardRule{
			{Table: "player", ShardCount: shardTestCount, SuffixFormat: "_%d"},
		},
	}
}

func UseMysqlDriverWithTableShard() error {
	c := mysqlConf()
	c.Mysql.Shard = playerTableShardConf()
	return drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeMySQL)),
		drivers.WithConfig(c),
	)
}

func UsePgsqlDriverWithTableShard() error {
	c := pgsqlConf()
	c.Pgsql.Shard = playerTableShardConf()
	return drivers.TryInit(context.Background(),
		drivers.WithDriverType(drivers.DriverTypePostgresSQL),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypePostgresSQL)),
		drivers.WithConfig(c),
	)
}
