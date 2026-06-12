package orm

import (
	logger "gitee.com/wxdqing/logger.git"
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
	Addr            string
	Name            string
	User            string
	Password        string
	MaxIdle         int
	MaxOpen         int
	MaxLifeTime     int
	MaxIdleLifeTime int
	Startup         GormStartupConf `json:"startup" mapstructure:"startup"`
	Shard           SQLShardConf    `json:"shard" mapstructure:"shard"`
}

type PgsqlConf struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxIdle         int
	MaxOpen         int
	MaxLifeTime     int
	MaxIdleLifeTime int
	Startup         GormStartupConf `json:"startup" mapstructure:"startup"`
	Shard           SQLShardConf    `json:"shard" mapstructure:"shard"`
}

type TcaplusConf struct {
	AppId     uint64
	ZoneId    uint32
	Addr      string
	Signature string
}
type RedisConf struct {
	Host     string
	Password string
	Index    int
}

type MongoConf struct {
	URI      string
	Database string // 逻辑库名，默认 orm；空时由驱动回退
}

//  加载配置文件

func init() {
	Config = Conf{
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
	if conf == nil {
		logger.Warnf("etcd conf is nil,use default conf")
		return nil
	}
	// 从db key下统一读取配置
	if dbConf, ok := conf["db"].(map[string]any); !ok {
		logger.Warnf("db conf is nil,use default conf")
	} else {
		// 读取driver
		if typ, ok := dbConf["driver"].(string); ok {
			Config.Driver = typ
		}

		// 读取mysql配置(可以直接在db下定义)
		err := mapstructure.Decode(dbConf, &Config.Mysql)
		if err != nil {
			logger.Panicf("decode mysql conf failed:%v", err)
		}
		applySQLConfDefaults(&Config.Mysql.Startup, &Config.Mysql.Shard, "mysql")

		// 兼容读取mysql配置(db.mysql)
		if mysqlConf, ok := dbConf["mysql"].(map[string]any); ok {
			err := mapstructure.Decode(mysqlConf, &Config.Mysql)
			if err != nil {
				logger.Panicf("decode mysql conf failed:%v", err)
			}
			applySQLConfDefaults(&Config.Mysql.Startup, &Config.Mysql.Shard, "mysql")
			logger.Warnf("db.mysql found in config,mysql config of db key(legacy) will be overrided")
		}
		// 读取redis配置(db.redis)
		if redisConf, ok := dbConf["redis"].(map[string]any); ok {
			err := mapstructure.Decode(redisConf, &Config.Redis)
			if err != nil {
				logger.Panicf("decode redis conf failed:%v", err)
			}
		} else {
			logger.Warnf("redis conf is nil,use default conf")
		}

		if mongoConf, ok := dbConf["mongo"].(map[string]any); ok {
			err := mapstructure.Decode(mongoConf, &Config.Mongo)
			if err != nil {
				logger.Panicf("decode mongo conf failed:%v", err)
			}
		}

		// 读取pgsql配置(db.pgsql)
		if pgsqlConf, ok := dbConf["pgsql"].(map[string]any); ok {
			err := mapstructure.Decode(pgsqlConf, &Config.Pgsql)
			if err != nil {
				logger.Panicf("decode pgsql conf failed:%v", err)
			}
			applySQLConfDefaults(&Config.Pgsql.Startup, &Config.Pgsql.Shard, "pgsql")
		}

		// 读取tcaplus配置(db.tcaplus)
		if tcaplusConf, ok := dbConf["tcaplus"].(map[string]any); ok {
			err := mapstructure.Decode(tcaplusConf, &Config.Tcaplus)
			if err != nil {
				logger.Panicf("decode tcaplus conf failed:%v", err)
			}
		} else {
			logger.Warnf("tcaplus conf is nil,use default conf")
		}
	}
	logger.Infof("db conf load success:%#v", Config)
	*c = Config
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
