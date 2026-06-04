package logic

import (
	"context"

	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest/metadata"
)

func UsePgsqlDriver() error {
	return drivers.TryInit(context.Background(),
		drivers.WithDriverType(drivers.DriverTypePostgresSQL),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypePostgresSQL)),
		drivers.WithConfig(pgsqlConf()),
	)
}

func UsePgsqlDriverWithNodeTables(nodeType string) error {
	return drivers.TryInit(context.Background(),
		drivers.WithDriverType(drivers.DriverTypePostgresSQL),
		drivers.WithNodeTables(metadata.GetNodeTables, nodeType),
		drivers.WithConfig(pgsqlConf()),
	)
}

