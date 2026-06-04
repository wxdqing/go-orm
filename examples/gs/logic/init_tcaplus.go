package logic

import (
	"context"

	"gs/pbtest/metadata"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
)

func UseTcaplusDriver() error {
	return drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeTcaplusDB)),
		drivers.WithConfig(&orm.Conf{
			Driver: drivers.DriverTypeTcaplusDB,
			Tcaplus: orm.TcaplusConf{
				AppId:     4,
				ZoneId:    1,
				Addr:      "tcp://192.168.1.243:9999",
				Signature: "CE3341761D417222",
			},
		}),
	)
}

func UseTcaplusDriverWithConfMap() error {
	return drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeTcaplusDB)),
		drivers.WithConfigMap(map[string]any{
			"db": map[string]interface{}{
				"driver": "tcaplus",
				"tcaplus": map[string]interface{}{
					"appId":     4,
					"zoneId":    1,
					"addr":      "tcp://192.168.1.243:9999",
					"signature": "CE3341761D417222",
				},
			},
		}),
	)
}
