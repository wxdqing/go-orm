package logic

import (
	"context"

	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest/metadata"
)

func UseMysqlDriver() error {
	return drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeMySQL)),
		drivers.WithConfig(mysqlConf()),
	)
}

func UseMysqlDriverWithHandlers() error {
	return drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeMySQL)),
		drivers.WithConfig(mysqlConf()),
		drivers.WithHandlerRegistry(ExampleHandlerRegistry()),
	)
}

func UseMysqlDriverWithConfMap() error {
	c := mysqlConf()
	return drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeMySQL)),
		drivers.WithConfigMap(map[string]any{
			"db": map[string]interface{}{
				"driver": "mysql",
				"mysql": map[string]interface{}{
					"addr":     c.Mysql.Addr,
					"name":     c.Mysql.Name,
					"user":     c.Mysql.User,
					"password": c.Mysql.Password,
				},
			},
		}),
	)
}

