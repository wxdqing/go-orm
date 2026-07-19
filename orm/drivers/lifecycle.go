package drivers

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/hook"
	"github.com/wxdqing/go-orm/orm/drivers/internal/kv"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	"github.com/wxdqing/go-orm/orm/drivers/internal/nop"
	sqldriver "github.com/wxdqing/go-orm/orm/drivers/internal/sql"
	"github.com/wxdqing/go-orm/orm/drivers/internal/tcaplus"
	"google.golang.org/protobuf/proto"
)

var lifecycle struct {
	mu     sync.Mutex
	ready  bool
	typ    DriverType
	inner  CoreDriver
	active Driver
	closed *closedDriver
}

func init() {
	lifecycle.closed = &closedDriver{}
	defaultDriver.set(lifecycle.closed)
}

type closedDriver struct{}

func (c *closedDriver) InitDB(context.Context, *driverapi.Options) error {
	return orm.ErrDbDriverNotInit
}
func (c *closedDriver) CloseDB(context.Context) error             { return nil }
func (c *closedDriver) Save(context.Context, proto.Message) error { return orm.ErrDbDriverNotInit }
func (c *closedDriver) Get(context.Context, proto.Message) error  { return orm.ErrDbDriverNotInit }
func (c *closedDriver) Find(context.Context, proto.Message) ([]proto.Message, error) {
	return nil, orm.ErrDbDriverNotInit
}
func (c *closedDriver) Delete(context.Context, proto.Message) error { return orm.ErrDbDriverNotInit }
func (c *closedDriver) Count(context.Context, proto.Message) (int64, error) {
	return 0, orm.ErrDbDriverNotInit
}
func (c *closedDriver) RunInTx(context.Context, func(context.Context) error) error {
	return orm.ErrDbDriverNotInit
}
func (c *closedDriver) Ping(context.Context) error { return orm.ErrDbDriverNotInit }

var _ driverapi.Driver = (*closedDriver)(nil)

func IsInitialized() bool {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	return lifecycle.ready
}

func CurrentDriverType() DriverType {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	if !lifecycle.ready {
		return ""
	}
	return lifecycle.typ
}

// slowInitHook is invoked after options are validated and before InitDB.
// Tests use it to hold the dial path without holding lifecycle.mu.
var slowInitHook func()

func TryInit(ctx context.Context, opts ...DriverOption) error {
	ctx = codec.EnsureCtx(ctx)
	o, err := buildDriverOptions(opts...)
	if err != nil {
		return err
	}

	for {
		lifecycle.mu.Lock()
		if lifecycle.ready {
			prev := detachLocked()
			lifecycle.mu.Unlock()
			if err := closePrevious(ctx, prev); err != nil {
				return fmt.Errorf("orm re-init close previous: %w", err)
			}
			continue
		}
		lifecycle.mu.Unlock()

		inner, err := newDriverForType(o.Type)
		if err != nil {
			return err
		}
		if slowInitHook != nil {
			slowInitHook()
		}
		apiOpts := o.opts()
		if err := inner.InitDB(ctx, apiOpts); err != nil {
			return errors.Join(err, inner.CloseDB(ctx))
		}

		lifecycle.mu.Lock()
		if lifecycle.ready {
			lifecycle.mu.Unlock()
			_ = inner.CloseDB(ctx)
			continue
		}
		meta.Init(o.Tables)
		syncMetaMaps()
		active := hook.Wrap(inner, string(o.Type), o.handlers)
		lifecycle.ready = true
		lifecycle.typ = o.Type
		lifecycle.inner = inner
		lifecycle.active = active
		defaultDriver.set(active)
		lifecycle.mu.Unlock()
		orm.GetLogger().Infof("orm init completed. driver type: %s", o.Type)
		return nil
	}
}

func Close(ctx context.Context) error {
	ctx = codec.EnsureCtx(ctx)
	lifecycle.mu.Lock()
	prev, ok := detachLockedOK()
	lifecycle.mu.Unlock()
	if !ok {
		return nil
	}
	return closePrevious(ctx, prev)
}

// detachLockedOK swaps the global driver to closed and clears lifecycle metadata.
// Caller must hold lifecycle.mu. Waiting for in-flight ops and CloseDB happen
// outside the lock so other goroutines can observe IsInitialized/Close progress.
func detachLockedOK() (*activeDriver, bool) {
	if !lifecycle.ready {
		defaultDriver.set(lifecycle.closed)
		return nil, false
	}
	prev := detachLocked()
	return prev, true
}

func detachLocked() *activeDriver {
	prev := defaultDriver.replace(lifecycle.closed)
	lifecycle.ready = false
	lifecycle.typ = ""
	lifecycle.inner = nil
	lifecycle.active = nil
	meta.Reset()
	syncMetaMaps()
	return prev
}

func Ping(ctx context.Context) error {
	return DefaultDbDriver.Ping(codec.EnsureCtx(ctx))
}

func buildDriverOptions(opts ...DriverOption) (*DriverOptions, error) {
	o := &DriverOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(o)
	}
	if o.err != nil {
		return nil, o.err
	}
	if o.Type == "" {
		if o.Conf != nil && o.Conf.Driver != "" {
			o.Type = DriverType(o.Conf.Driver)
		} else {
			o.Type = DriverTypeMySQL
		}
	}
	if err := validateDriverOptions(o); err != nil {
		return nil, err
	}
	return o, nil
}

func syncMetaMaps() {
	DbTableNameMapping = meta.DbTableNameMapping
	ValueNameMapping = meta.ValueNameMapping
}

func validateDriverOptions(o *DriverOptions) error {
	if o.Conf == nil {
		return fmt.Errorf("%w: database conf is required (WithConfig / WithConfigMap)", orm.ErrInvalidDriverOptions)
	}
	if len(o.Tables) == 0 {
		return fmt.Errorf("%w: at least one table is required (WithTables)", orm.ErrInvalidDriverOptions)
	}
	switch o.Type {
	case DriverTypeNop, DriverTypeMySQL, DriverTypePostgresSQL, DriverTypeTcaplusDB, DriverTypeRedis, DriverTypeMongoDB:
		return nil
	default:
		return fmt.Errorf("%w: unsupported driver %s", orm.ErrInvalidDriverOptions, o.Type)
	}
}

func newDriverForType(t DriverType) (CoreDriver, error) {
	switch t {
	case DriverTypeNop:
		return nop.New(), nil
	case DriverTypeMySQL:
		return sqldriver.NewMySQL(), nil
	case DriverTypeRedis:
		return kv.NewRedis(), nil
	case DriverTypeTcaplusDB:
		return tcaplus.New(), nil
	case DriverTypeMongoDB:
		return kv.NewMongo(), nil
	case DriverTypePostgresSQL:
		return sqldriver.NewPgsql(), nil
	default:
		return nil, fmt.Errorf("%w: %s", orm.ErrInvalidDriverOptions, t)
	}
}

// 兼容测试：表元数据映射（由 meta 包维护）。
var (
	DbTableNameMapping = meta.DbTableNameMapping
	ValueNameMapping   = meta.ValueNameMapping
)

// 兼容：部分测试/工具直接引用。
type TableMetaData = meta.TableMetaData

func GetMetaByValue(value proto.Message) *meta.TableMetaData {
	return meta.GetMetaByValue(value)
}

func GetTableName(p proto.Message) orm.TableName {
	return meta.GetTableName(p)
}
