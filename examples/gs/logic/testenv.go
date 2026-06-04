package logic

import (
	"os"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"github.com/wxdqing/go-orm/testenv"
)

// 与 orm/drivers/testenv_test.go 默认一致，可用环境变量覆盖（见 docs/orm/README.md）。

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mysqlConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypeMySQL,
		Mysql: orm.MysqlConf{
			Addr:     envOr("ORM_TEST_MYSQL_ADDR", "127.0.0.1:3306"),
			Name:     envOr("ORM_TEST_MYSQL_DB", "game"),
			User:     envOr("ORM_TEST_MYSQL_USER", "root"),
			Password: envOr("ORM_TEST_MYSQL_PASSWORD", "root123"),
			Startup:  orm.DefaultGormStartup("mysql"),
		},
	}
}

func pgsqlConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypePostgresSQL,
		Pgsql: orm.PgsqlConf{
			Host:     envOr("ORM_TEST_PGSQL_HOST", "127.0.0.1"),
			Port:     envOr("ORM_TEST_PGSQL_PORT", "5432"),
			Name:     envOr("ORM_TEST_PGSQL_DB", "game"),
			User:     envOr("ORM_TEST_PGSQL_USER", "postgres"),
			Password: envOr("ORM_TEST_PGSQL_PASSWORD", "postgres123"),
			Startup:  orm.DefaultGormStartup("pgsql"),
		},
	}
}

func redisConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypeRedis,
		Redis: orm.RedisConf{
			Host:     envOr("ORM_TEST_REDIS_ADDR", "127.0.0.1:16379"),
			Password: os.Getenv("ORM_TEST_REDIS_PASSWORD"),
			Index:    0,
		},
	}
}

func mongoConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypeMongoDB,
		Mongo: orm.MongoConf{
			URI:      testenv.MongoURI(),
			Database: testenv.MongoDatabase(),
		},
	}
}
