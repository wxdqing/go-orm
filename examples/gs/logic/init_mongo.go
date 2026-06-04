package logic

import (
	"context"

	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest/metadata"
)

func UseMongoDriver() error {
	return drivers.TryInit(context.Background(),
		drivers.WithDriverType(drivers.DriverTypeMongoDB),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeMongoDB)),
		drivers.WithConfig(mongoConf()),
	)
}
