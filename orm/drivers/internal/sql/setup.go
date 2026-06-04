package sql

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/logger_ext"
	logger "git.wxdqing.com/sprout/logger.git"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

func buildGormConfig(startup orm.GormStartupConf) *gorm.Config {
	return &gorm.Config{
		SkipDefaultTransaction: startup.SkipDefaultTransaction,
		PrepareStmt:            startup.PrepareStmt,
		DisableAutomaticPing:   startup.DisableAutomaticPing,
		Logger:                 logger_ext.NewDbLogger(),
	}
}

func mysqlDSN(c orm.MysqlConf) string {
	dbConf := c
	q := url.Values{}
	q.Set("charset", "utf8mb4")
	q.Set("parseTime", "True")
	if dbConf.Startup.TimeZone != "" {
		q.Set("loc", dbConf.Startup.TimeZone)
	} else {
		q.Set("loc", "Local")
	}
	q.Set("tls", "false")
	for k, v := range dbConf.Startup.ExtraDSN {
		q.Set(k, v)
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s",
		dbConf.User, dbConf.Password, dbConf.Addr, dbConf.Name, q.Encode())
}

func pgsqlDSN(c orm.PgsqlConf) string {
	dbConf := c
	ssl := dbConf.Startup.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	tz := dbConf.Startup.TimeZone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	parts := []string{
		fmt.Sprintf("host=%s", dbConf.Host),
		fmt.Sprintf("user=%s", dbConf.User),
		fmt.Sprintf("password=%s", dbConf.Password),
		fmt.Sprintf("dbname=%s", dbConf.Name),
		fmt.Sprintf("port=%s", dbConf.Port),
		fmt.Sprintf("sslmode=%s", ssl),
		fmt.Sprintf("TimeZone=%s", tz),
	}
	for k, v := range dbConf.Startup.ExtraDSN {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

func openMysqlDB(c orm.MysqlConf) (*gorm.DB, error) {
	startup := c.Startup
	if startup.TimeZone == "" && startup.TableOptions == "" {
		startup = orm.DefaultGormStartup("mysql")
		c.Startup = startup
	}
	return gorm.Open(mysql.New(mysql.Config{
		DSN:                       mysqlDSN(c),
		SkipInitializeWithVersion: false,
	}), buildGormConfig(startup))
}

func openPgsqlDB(c orm.PgsqlConf) (*gorm.DB, error) {
	startup := c.Startup
	if startup.TimeZone == "" {
		startup = orm.DefaultGormStartup("pgsql")
		c.Startup = startup
	}
	return gorm.Open(postgres.Open(pgsqlDSN(c)), buildGormConfig(startup))
}

func openDatabaseShardDBs(driver driverapi.Type, mysqlC orm.MysqlConf, pgsqlC orm.PgsqlConf, shard orm.SQLShardConf) ([]*gorm.DB, error) {
	if shard.Mode != orm.ShardModeDatabase || len(shard.Sources) == 0 {
		return nil, nil
	}
	dbs := make([]*gorm.DB, 0, len(shard.Sources))
	for i, src := range shard.Sources {
		var (
			db  *gorm.DB
			err error
		)
		switch driver {
		case driverapi.TypeMySQL:
			mc := mysqlC
			if src.Addr != "" {
				mc.Addr = src.Addr
			}
			if src.User != "" {
				mc.User = src.User
			}
			if src.Password != "" {
				mc.Password = src.Password
			}
			if src.DBName != "" {
				mc.Name = src.DBName
			}
			db, err = openMysqlDB(mc)
		case driverapi.TypePostgresSQL:
			pc := pgsqlC
			if src.Host != "" {
				pc.Host = src.Host
			}
			if src.Port != "" {
				pc.Port = src.Port
			}
			if src.User != "" {
				pc.User = src.User
			}
			if src.Password != "" {
				pc.Password = src.Password
			}
			if src.DBName != "" {
				pc.Name = src.DBName
			}
			db, err = openPgsqlDB(pc)
		default:
			return nil, fmt.Errorf("unsupported shard driver: %s", driver)
		}
		if err != nil {
			return nil, fmt.Errorf("open shard source %d (%s): %w", i, src.Name, err)
		}
		dbs = append(dbs, db)
		logger.Infof("orm shard database source ready: index=%d name=%s", i, src.Name)
	}
	return dbs, nil
}

func poolSettings(o *driverapi.Options) (maxIdle, maxOpen, maxLifetime, maxIdleLifetime int) {
	maxIdle, maxOpen, maxLifetime, maxIdleLifetime = 50, 500, 7200, 7200
	if o.Type == driverapi.TypePostgresSQL || o.Conf.Driver == string(driverapi.TypePostgresSQL) {
		if o.Conf.Pgsql.MaxIdle != 0 {
			maxIdle = o.Conf.Pgsql.MaxIdle
		}
		if o.Conf.Pgsql.MaxOpen != 0 {
			maxOpen = o.Conf.Pgsql.MaxOpen
		}
		if o.Conf.Pgsql.MaxLifeTime != 0 {
			maxLifetime = o.Conf.Pgsql.MaxLifeTime
		}
		if o.Conf.Pgsql.MaxIdleLifeTime != 0 {
			maxIdleLifetime = o.Conf.Pgsql.MaxIdleLifeTime
		}
		return
	}
	if o.Conf.Mysql.MaxIdle != 0 {
		maxIdle = o.Conf.Mysql.MaxIdle
	}
	if o.Conf.Mysql.MaxOpen != 0 {
		maxOpen = o.Conf.Mysql.MaxOpen
	}
	if o.Conf.Mysql.MaxLifeTime != 0 {
		maxLifetime = o.Conf.Mysql.MaxLifeTime
	}
	if o.Conf.Mysql.MaxIdleLifeTime != 0 {
		maxIdleLifetime = o.Conf.Mysql.MaxIdleLifeTime
	}
	return
}

func applyDBResolver(db *gorm.DB, o *driverapi.Options) error {
	maxIdle, maxOpen, maxLifetime, maxIdleLifetime := poolSettings(o)
	return db.Use(dbresolver.Register(dbresolver.Config{}).
		SetConnMaxIdleTime(time.Duration(maxIdleLifetime) * time.Second).
		SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second).
		SetMaxIdleConns(maxIdle).
		SetMaxOpenConns(maxOpen))
}

// applyPoolToDB 对单个 *gorm.DB 设置连接池（分库实例也需调用）。
func applyPoolToDB(db *gorm.DB, o *driverapi.Options) error {
	if db == nil {
		return nil
	}
	maxIdle, maxOpen, maxLifetime, maxIdleLifetime := poolSettings(o)
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)
	sqlDB.SetConnMaxIdleTime(time.Duration(maxIdleLifetime) * time.Second)
	return nil
}

func closeGormDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func sqlShardConf(o *driverapi.Options) orm.SQLShardConf {
	if o.Type == driverapi.TypePostgresSQL || o.Conf.Driver == string(driverapi.TypePostgresSQL) {
		return o.Conf.Pgsql.Shard
	}
	return o.Conf.Mysql.Shard
}

func gormStartup(o *driverapi.Options) orm.GormStartupConf {
	if o.Type == driverapi.TypePostgresSQL || o.Conf.Driver == string(driverapi.TypePostgresSQL) {
		s := o.Conf.Pgsql.Startup
		if s.TimeZone == "" {
			s = orm.DefaultGormStartup("pgsql")
		}
		return s
	}
	s := o.Conf.Mysql.Startup
	if s.TimeZone == "" && s.TableOptions == "" {
		s = orm.DefaultGormStartup("mysql")
	}
	return s
}
