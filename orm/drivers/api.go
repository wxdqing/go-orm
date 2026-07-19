package drivers

import (
	"fmt"
	"reflect"

	"github.com/redis/go-redis/v9"
	tcapluspb "github.com/tencentyun/tcaplusdb-go-sdk/pb"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/hook"
	"github.com/wxdqing/go-orm/orm/drivers/internal/kv"
	"github.com/wxdqing/go-orm/orm/drivers/internal/nop"
	"github.com/wxdqing/go-orm/orm/drivers/internal/sql"
	"github.com/wxdqing/go-orm/orm/drivers/internal/tcaplus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type (
	Driver       = driverapi.Driver
	CoreDriver   = driverapi.CoreDriver
	FullDriver   = driverapi.FullDriver
	Finder       = driverapi.Finder
	Counter      = driverapi.Counter
	Transactor   = driverapi.Transactor
	Pinger       = driverapi.Pinger
	SQLQuerier   = driverapi.SQLQuerier
	SQLDriver    = driverapi.SQLDriver
	Rows         = driverapi.Rows
	DriverType   = driverapi.Type
	DriverOption func(o *DriverOptions)

	DriverOptions struct {
		Type     DriverType
		Conf     *orm.Conf
		Tables   []proto.Message
		handlers *hook.HandlerRegistry
		err      error
	}

	HandlerRegistry = hook.HandlerRegistry
	HandlerFuncs    = hook.HandlerFuncs

	MysqlDriver     = sql.MySQL
	PgsqlDriver     = sql.Pgsql
	TcaplusDbDriver = tcaplus.Driver
)

const (
	DriverTypeNop         = driverapi.TypeNop
	DriverTypeMySQL       = driverapi.TypeMySQL
	DriverTypeRedis       = driverapi.TypeRedis
	DriverTypeTcaplusDB   = driverapi.TypeTcaplusDB
	DriverTypeMongoDB     = driverapi.TypeMongoDB
	DriverTypePostgresSQL = driverapi.TypePostgresSQL
)

var DefaultDbDriver Driver = defaultDriver

var defaultDriver = &driverProxy{}

func (o *DriverOptions) opts() *driverapi.Options {
	if o == nil {
		return nil
	}
	return &driverapi.Options{Type: o.Type, Conf: o.Conf, Tables: o.Tables}
}

func WithConfig(conf *orm.Conf) DriverOption {
	return func(o *DriverOptions) {
		if conf == nil {
			o.err = fmt.Errorf("%w: database conf is nil", orm.ErrInvalidDriverOptions)
			return
		}
		o.Conf = conf
		o.Type = DriverType(conf.Driver)
	}
}

func WithConfigMap(conf map[string]any) DriverOption {
	return func(o *DriverOptions) {
		c := &orm.Conf{}
		if err := orm.DecodeMapToStruct(conf, c); err != nil {
			o.err = fmt.Errorf("%w: decode config map: %v", orm.ErrInvalidDriverOptions, err)
			return
		}
		o.Conf = c
		o.Type = DriverType(c.Driver)
	}
}

func WithTablesByDriverType(getTables func(dbType string) []proto.Message) DriverOption {
	return func(o *DriverOptions) {
		if getTables != nil {
			if o.Type != "" {
				o.Tables = append(o.Tables, getTables(string(o.Type))...)
			} else {
				o.Tables = append(o.Tables, getTables(string(DriverTypeMySQL))...)
			}
		}
	}
}

func WithTables(tables []proto.Message) DriverOption {
	return func(o *DriverOptions) {
		o.Tables = append(o.Tables, tables...)
	}
}

func WithNodeTables(getNodeTables func(dbType, nodeType string) []proto.Message, nodeType string) DriverOption {
	return func(o *DriverOptions) {
		if getNodeTables == nil || nodeType == "" {
			return
		}
		dbType := string(o.Type)
		if dbType == "" {
			dbType = string(DriverTypeMySQL)
		}
		o.Tables = append(o.Tables, getNodeTables(dbType, nodeType)...)
	}
}

func WithDriverType(typ DriverType) DriverOption {
	return func(o *DriverOptions) {
		o.Type = typ
	}
}

func ExcludeTables(tb ...proto.Message) DriverOption {
	return func(o *DriverOptions) {
		excludeMap := make(map[string]struct{})
		for _, t := range tb {
			tbName := reflect.TypeOf(t).Elem().Name()
			excludeMap[tbName] = struct{}{}
		}
		filtered := make([]proto.Message, 0, len(o.Tables))
		for _, t := range o.Tables {
			tbName := reflect.TypeOf(t).Elem().Name()
			if _, excluded := excludeMap[tbName]; !excluded {
				filtered = append(filtered, t)
			}
		}
		o.Tables = filtered
	}
}

func DefaultHandlerRegistry() *HandlerRegistry {
	return hook.DefaultRegistry()
}

func RegisterHandlerFuncs(f HandlerFuncs) {
	hook.DefaultRegistry().Register(orm.TableName(f.Table), f)
}

func WithHandlerRegistry(r *HandlerRegistry) DriverOption {
	return func(o *DriverOptions) {
		o.handlers = r
	}
}

func WithHandlers(handlers ...orm.TableHandler) DriverOption {
	return func(o *DriverOptions) {
		reg := o.handlers
		if reg == nil {
			reg = hook.DefaultRegistry()
		}
		for _, h := range handlers {
			table := ""
			if t, ok := h.(interface{ Table() string }); ok {
				table = t.Table()
			}
			reg.Register(orm.TableName(table), h)
		}
		o.handlers = reg
	}
}

func WithHandlerFuncs(funcs ...HandlerFuncs) DriverOption {
	return func(o *DriverOptions) {
		reg := o.handlers
		if reg == nil {
			reg = hook.DefaultRegistry()
		}
		for _, f := range funcs {
			reg.Register(orm.TableName(f.Table), f)
		}
		o.handlers = reg
	}
}

func ToGorm() *gorm.DB {
	return DriverGorm(DefaultDbDriver)
}

func DriverGorm(d CoreDriver) *gorm.DB {
	d = unwrapDriver(d)
	switch x := d.(type) {
	case *sql.MySQL:
		return x.GormDB()
	case *sql.Pgsql:
		return x.GormDB()
	case interface{ GormDB() *gorm.DB }:
		return x.GormDB()
	}
	return nil
}

func ToRedis() *redis.Client {
	return DriverRedis(DefaultDbDriver)
}

func DriverRedis(d CoreDriver) *redis.Client {
	d = unwrapDriver(d)
	if r, ok := d.(*kv.Redis); ok {
		return r.Client()
	}
	return nil
}

func ToMongo() *mongo.Client {
	return DriverMongo(DefaultDbDriver)
}

func DriverMongo(d CoreDriver) *mongo.Client {
	d = unwrapDriver(d)
	if m, ok := d.(*kv.Mongo); ok {
		return m.Client()
	}
	return nil
}

func ToTcaplusClient() *tcapluspb.PBClient {
	if td, ok := unwrapDriver(DefaultDbDriver).(*tcaplus.Driver); ok {
		return td.Cli
	}
	return nil
}

func unwrapDriver(d CoreDriver) CoreDriver {
	for d != nil {
		unwrapper, ok := d.(interface{ Unwrap() CoreDriver })
		if !ok {
			return d
		}
		next := unwrapper.Unwrap()
		if next == d {
			return d
		}
		d = next
	}
	return nil
}

var (
	NewMySQLDriver       = sql.NewMySQL
	NewPostgresSQLDriver = sql.NewPgsql
	NewRedisDriver       = kv.NewRedis
	NewMongoDBDriver     = kv.NewMongo
	NewTcaplusDbDriver   = tcaplus.New
	NewNopDriver         = nop.New
)
