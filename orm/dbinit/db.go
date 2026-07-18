// Package dbinit provides database connection lifecycle (open / close / ping)
// without ORM table registration or handler wiring.
//
// Use drivers.TryInit when you need proto CRUD, AutoMigrate, and hooks.
// Use dbinit when you only need a raw client (GORM, Redis, Mongo).
package dbinit

import (
	"context"
	"fmt"
	"sync"

	logger "gitee.com/wxdqing/logger.git"
	"github.com/redis/go-redis/v9"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

const (
	TypeMySQL       = driverapi.TypeMySQL
	TypePostgresSQL = driverapi.TypePostgresSQL
	TypeRedis       = driverapi.TypeRedis
	TypeMongoDB     = driverapi.TypeMongoDB
)

// DB holds one initialized connection. Safe for concurrent Ping/Close;
// accessors return the underlying client which follows its own concurrency rules.
type DB struct {
	typ    driverapi.Type
	driver driverapi.Driver
}

// Open connects using typ and conf. conf must contain the section for typ
// (Mysql / Pgsql / Redis / Mongo). Tables and handlers are not required.
func Open(ctx context.Context, typ driverapi.Type, conf *orm.Conf) (*DB, error) {
	if err := validateConf(typ, conf); err != nil {
		return nil, err
	}
	driver, err := newDriver(typ)
	if err != nil {
		return nil, err
	}
	c := cloneConf(conf, typ)
	opts := &driverapi.Options{Type: typ, Conf: c}
	if err := driver.InitDB(ensureCtx(ctx), opts); err != nil {
		return nil, err
	}
	logger.Infof("dbinit open completed. driver type: %s", typ)
	return &DB{typ: typ, driver: driver}, nil
}

func OpenMySQL(ctx context.Context, conf *orm.Conf) (*DB, error) {
	return Open(ctx, TypeMySQL, conf)
}

func OpenPostgres(ctx context.Context, conf *orm.Conf) (*DB, error) {
	return Open(ctx, TypePostgresSQL, conf)
}

func OpenRedis(ctx context.Context, conf *orm.Conf) (*DB, error) {
	return Open(ctx, TypeRedis, conf)
}

func OpenMongo(ctx context.Context, conf *orm.Conf) (*DB, error) {
	return Open(ctx, TypeMongoDB, conf)
}

func (db *DB) Type() driverapi.Type {
	if db == nil {
		return ""
	}
	return db.typ
}

func (db *DB) Driver() driverapi.Driver {
	if db == nil {
		return nil
	}
	return db.driver
}

func (db *DB) Gorm() *gorm.DB {
	return drivers.DriverGorm(db.driver)
}

func (db *DB) Redis() *redis.Client {
	return drivers.DriverRedis(db.driver)
}

func (db *DB) Mongo() *mongo.Client {
	return drivers.DriverMongo(db.driver)
}

func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.driver == nil {
		return orm.ErrDbDriverNotInit
	}
	return db.driver.Ping(ensureCtx(ctx))
}

func (db *DB) Close(ctx context.Context) error {
	if db == nil || db.driver == nil {
		return nil
	}
	err := db.driver.CloseDB(ensureCtx(ctx))
	db.driver = nil
	return err
}

// Group owns multiple DB handles and closes them in reverse init order (LIFO).
type Group struct {
	mu  sync.Mutex
	dbs []*DB
}

func (g *Group) Add(db *DB) {
	if g == nil || db == nil {
		return
	}
	g.mu.Lock()
	g.dbs = append(g.dbs, db)
	g.mu.Unlock()
}

func (g *Group) Close(ctx context.Context) error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	dbs := g.dbs
	g.dbs = nil
	g.mu.Unlock()
	var first error
	for i := len(dbs) - 1; i >= 0; i-- {
		if err := dbs[i].Close(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newDriver(typ driverapi.Type) (driverapi.Driver, error) {
	switch typ {
	case TypeMySQL:
		return drivers.NewMySQLDriver(), nil
	case TypePostgresSQL:
		return drivers.NewPostgresSQLDriver(), nil
	case TypeRedis:
		return drivers.NewRedisDriver(), nil
	case TypeMongoDB:
		return drivers.NewMongoDBDriver(), nil
	default:
		return nil, fmt.Errorf("%w: unsupported dbinit type %s", orm.ErrInvalidDriverOptions, typ)
	}
}

func validateConf(typ driverapi.Type, conf *orm.Conf) error {
	if conf == nil {
		return fmt.Errorf("%w: conf is required", orm.ErrInvalidDriverOptions)
	}
	switch typ {
	case TypeMySQL:
		if conf.Mysql.Addr == "" {
			return fmt.Errorf("%w: mysql addr is required", orm.ErrInvalidDriverOptions)
		}
	case TypePostgresSQL:
		if conf.Pgsql.Host == "" {
			return fmt.Errorf("%w: pgsql host is required", orm.ErrInvalidDriverOptions)
		}
	case TypeRedis:
		if conf.Redis.Host == "" {
			return fmt.Errorf("%w: redis host is required", orm.ErrInvalidDriverOptions)
		}
	case TypeMongoDB:
		return nil
	default:
		return fmt.Errorf("%w: unsupported dbinit type %s", orm.ErrInvalidDriverOptions, typ)
	}
	return nil
}

func cloneConf(conf *orm.Conf, typ driverapi.Type) *orm.Conf {
	c := *conf
	if c.Driver == "" {
		c.Driver = string(typ)
	}
	return &c
}

func ensureCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
