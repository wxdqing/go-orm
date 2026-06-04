package logic

import (
	"context"

	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest/metadata"
)

func UseRedisDriver() error {
	return drivers.TryInit(context.Background(),
		drivers.WithDriverType(drivers.DriverTypeRedis),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeRedis)),
		drivers.WithConfig(redisConf()),
	)
}
