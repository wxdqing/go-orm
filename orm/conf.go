package orm

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

var Config Conf

// Conf  对应配置文件中的[db]部分
type Conf struct {
	Driver  string      // toml: [db]driver:mysql/tcaplus/redis... map: "db"."driver"
	Mysql   MysqlConf   // toml: [db.mysql] (higher priority) or [db] (legacy) map: "db"."mysql" or "db"
	Pgsql   PgsqlConf   // toml: [db.pgsql] map: "db"."pgsql"
	Tcaplus TcaplusConf // toml: [db.tcaplus] map: "db"."tcaplus"
	Redis   RedisConf   // toml: [db.redis] map: "db"."redis"
	Mongo   MongoConf   // toml: [db.mongo] map: "db"."mongo"
}

type MysqlConf struct {
	Addr            string          `mapstructure:"addr"`
	Name            string          `mapstructure:"name"`
	User            string          `mapstructure:"user"`
	Password        string          `mapstructure:"password"`
	MaxIdle         int             `mapstructure:"max_idle"`
	MaxOpen         int             `mapstructure:"max_open"`
	MaxLifeTime     int             `mapstructure:"max_life_time"`
	MaxIdleLifeTime int             `mapstructure:"max_idle_life_time"`
	Startup         GormStartupConf `json:"startup" mapstructure:"startup"`
	Shard           SQLShardConf    `json:"shard" mapstructure:"shard"`
}

type PgsqlConf struct {
	Host            string          `mapstructure:"host"`
	Port            string          `mapstructure:"port"`
	Name            string          `mapstructure:"name"`
	User            string          `mapstructure:"user"`
	Password        string          `mapstructure:"password"`
	MaxIdle         int             `mapstructure:"max_idle"`
	MaxOpen         int             `mapstructure:"max_open"`
	MaxLifeTime     int             `mapstructure:"max_life_time"`
	MaxIdleLifeTime int             `mapstructure:"max_idle_life_time"`
	Startup         GormStartupConf `json:"startup" mapstructure:"startup"`
	Shard           SQLShardConf    `json:"shard" mapstructure:"shard"`
}

type TcaplusConf struct {
	AppId     uint64 `mapstructure:"app_id"`
	ZoneId    uint32 `mapstructure:"zone_id"`
	Addr      string `mapstructure:"addr"`
	Signature string `mapstructure:"signature"`
}
type RedisConf struct {
	Host     string `mapstructure:"host"`
	Password string `mapstructure:"password"`
	Index    int    `mapstructure:"index"`
	Cluster  bool   `mapstructure:"cluster"`
}

type MongoConf struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"` // 逻辑库名，默认 orm；空时由驱动回退
}

//  加载配置文件

func init() {
	Config = DefaultConf()
}

func DefaultConf() Conf {
	return Conf{
		Driver: "mysql",
		Mysql: MysqlConf{
			Addr:            "localhost:3306",
			Name:            "default-orm",
			User:            "root",
			Password:        "root",
			MaxIdle:         10,
			MaxOpen:         100,
			MaxLifeTime:     3600,
			MaxIdleLifeTime: 3600,
			Startup:         DefaultGormStartup("mysql"),
		},
		Redis: RedisConf{
			Host:     "localhost:16379",
			Password: "",
			Index:    0,
		},
		Mongo: MongoConf{
			URI: "mongodb://localhost:27017",
		},
		Pgsql: PgsqlConf{
			Host:            "localhost",
			Port:            "5432",
			Name:            "game",
			User:            "postgres",
			Password:        "postgres",
			MaxIdle:         10,
			MaxOpen:         100,
			MaxLifeTime:     3600,
			MaxIdleLifeTime: 3600,
			Startup:         DefaultGormStartup("pgsql"),
		},
		Tcaplus: TcaplusConf{
			AppId:     0,
			ZoneId:    0,
			Addr:      "tcp://tcaplusdb.tencentcloudapi.com:443",
			Signature: "",
		},
	}
}

func DecodeMapToStruct(conf map[string]any, c *Conf) error {
	if c == nil {
		return fmt.Errorf("orm config target is nil")
	}
	decoded := DefaultConf()
	if conf == nil {
		GetLogger().Warnf("etcd conf is nil,use default conf")
		*c = decoded
		return nil
	}
	// 从db key下统一读取配置
	if dbConf, ok := conf["db"].(map[string]any); !ok {
		GetLogger().Warnf("db conf is nil,use default conf")
	} else {
		// 读取driver
		if typ, ok := dbConf["driver"].(string); ok {
			decoded.Driver = typ
		}

		// 读取mysql配置(可以直接在db下定义)
		err := mapstructure.Decode(dbConf, &decoded.Mysql)
		if err != nil {
			return fmt.Errorf("decode mysql conf: %w", err)
		}
		applySQLConfDefaults(&decoded.Mysql.Startup, &decoded.Mysql.Shard, "mysql")

		// 兼容读取mysql配置(db.mysql)
		if mysqlConf, ok := dbConf["mysql"].(map[string]any); ok {
			err := mapstructure.Decode(mysqlConf, &decoded.Mysql)
			if err != nil {
				return fmt.Errorf("decode mysql conf: %w", err)
			}
			applySQLConfDefaults(&decoded.Mysql.Startup, &decoded.Mysql.Shard, "mysql")
			GetLogger().Warnf("db.mysql found in config,mysql config of db key(legacy) will be overrided")
		}
		// 读取redis配置(db.redis)
		if redisConf, ok := dbConf["redis"].(map[string]any); ok {
			err := mapstructure.Decode(redisConf, &decoded.Redis)
			if err != nil {
				return fmt.Errorf("decode redis conf: %w", err)
			}
		} else {
			GetLogger().Warnf("redis conf is nil,use default conf")
		}

		if mongoConf, ok := dbConf["mongo"].(map[string]any); ok {
			err := mapstructure.Decode(mongoConf, &decoded.Mongo)
			if err != nil {
				return fmt.Errorf("decode mongo conf: %w", err)
			}
		}

		// 读取pgsql配置(db.pgsql)
		if pgsqlConf, ok := dbConf["pgsql"].(map[string]any); ok {
			err := mapstructure.Decode(pgsqlConf, &decoded.Pgsql)
			if err != nil {
				return fmt.Errorf("decode pgsql conf: %w", err)
			}
			applySQLConfDefaults(&decoded.Pgsql.Startup, &decoded.Pgsql.Shard, "pgsql")
		}

		// 读取tcaplus配置(db.tcaplus)
		if tcaplusConf, ok := dbConf["tcaplus"].(map[string]any); ok {
			err := mapstructure.Decode(tcaplusConf, &decoded.Tcaplus)
			if err != nil {
				return fmt.Errorf("decode tcaplus conf: %w", err)
			}
		} else {
			GetLogger().Warnf("tcaplus conf is nil,use default conf")
		}
	}
	GetLogger().Infof("db conf load success")
	*c = decoded
	return nil
}

func applySQLConfDefaults(startup *GormStartupConf, shard *SQLShardConf, driver string) {
	def := DefaultGormStartup(driver)
	if startup.TimeZone == "" {
		startup.TimeZone = def.TimeZone
	}
	if startup.SSLMode == "" {
		startup.SSLMode = def.SSLMode
	}
	if driver == "mysql" && startup.TableOptions == "" {
		startup.TableOptions = def.TableOptions
	}
	if !startup.SkipDefaultTransaction && !startup.PrepareStmt {
		startup.SkipDefaultTransaction = def.SkipDefaultTransaction
		startup.PrepareStmt = def.PrepareStmt
	}
	shard.Normalize()
}
