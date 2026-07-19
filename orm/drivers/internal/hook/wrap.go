package hook

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"google.golang.org/protobuf/proto"
)

type driver struct {
	inner   driverapi.CoreDriver
	driver  string
	handler *HandlerRegistry
}

func (h *driver) Unwrap() driverapi.CoreDriver {
	return h.inner
}

// Wrap 在内部 Driver 外包装外部处理器。
func Wrap(inner driverapi.CoreDriver, driverType string, reg *HandlerRegistry) driverapi.Driver {
	if reg == nil {
		reg = DefaultRegistry()
	}
	if !reg.hasHandlers() {
		if full, ok := inner.(driverapi.Driver); ok {
			return full
		}
	}
	return &driver{
		inner:   inner,
		driver:  driverType,
		handler: reg,
	}
}

func (r *HandlerRegistry) hasHandlers() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries) > 0 || len(r.fallback) > 0
}

func (h *driver) InitDB(ctx context.Context, o *driverapi.Options) error {
	dctx := &orm.DriverContext{Ctx: codec.EnsureCtx(ctx), Op: orm.OpInitDB, Driver: h.driver}
	if handled, err := h.handler.initDB(dctx); handled {
		return err
	}
	if err := h.handler.before(dctx, nil); err != nil {
		return err
	}
	err := h.inner.InitDB(ctx, o)
	return errors.Join(err, h.handler.after(dctx, nil, err))
}

func (h *driver) Save(ctx context.Context, tb proto.Message) error {
	dctx := newDriverContext(ctx, orm.OpSave, h.driver, tb)
	if handled, err := h.handler.save(dctx, tb); handled {
		return err
	}
	if err := h.handler.before(dctx, tb); err != nil {
		return err
	}
	err := h.inner.Save(ctx, tb)
	return errors.Join(err, h.handler.after(dctx, tb, err))
}

func (h *driver) Get(ctx context.Context, tb proto.Message) error {
	dctx := newDriverContext(ctx, orm.OpGet, h.driver, tb)
	if handled, err := h.handler.get(dctx, tb); handled {
		return err
	}
	if err := h.handler.before(dctx, tb); err != nil {
		return err
	}
	err := h.inner.Get(ctx, tb)
	return errors.Join(err, h.handler.after(dctx, tb, err))
}

func (h *driver) Find(ctx context.Context, cond proto.Message) ([]proto.Message, error) {
	dctx := newDriverContext(ctx, orm.OpFind, h.driver, cond)
	dctx.Condition = cond
	if result, handled, err := h.handler.find(dctx, cond); handled {
		return result, err
	}
	if err := h.handler.before(dctx, cond); err != nil {
		return nil, err
	}
	finder, ok := h.inner.(driverapi.Finder)
	if !ok {
		return nil, orm.ErrNotImplemented
	}
	result, err := finder.Find(ctx, cond)
	return result, errors.Join(err, h.handler.after(dctx, cond, err))
}

func (h *driver) CloseDB(ctx context.Context) error {
	dctx := &orm.DriverContext{Ctx: codec.EnsureCtx(ctx), Op: orm.OpCloseDB, Driver: h.driver}
	if handled, err := h.handler.closeDB(dctx); handled {
		return err
	}
	if err := h.handler.before(dctx, nil); err != nil {
		return err
	}
	err := h.inner.CloseDB(ctx)
	return errors.Join(err, h.handler.after(dctx, nil, err))
}

func (h *driver) Delete(ctx context.Context, tb proto.Message) error {
	dctx := newDriverContext(ctx, orm.OpDelete, h.driver, tb)
	if handled, err := h.handler.delete(dctx, tb); handled {
		return err
	}
	if err := h.handler.before(dctx, tb); err != nil {
		return err
	}
	err := h.inner.Delete(ctx, tb)
	return errors.Join(err, h.handler.after(dctx, tb, err))
}

func (h *driver) Count(ctx context.Context, cond proto.Message) (int64, error) {
	dctx := newDriverContext(ctx, orm.OpCount, h.driver, cond)
	dctx.Condition = cond
	if count, handled, err := h.handler.count(dctx, cond); handled {
		return count, err
	}
	if err := h.handler.before(dctx, cond); err != nil {
		return 0, err
	}
	counter, ok := h.inner.(driverapi.Counter)
	if !ok {
		return 0, orm.ErrNotImplemented
	}
	count, err := counter.Count(ctx, cond)
	return count, errors.Join(err, h.handler.after(dctx, cond, err))
}

func (h *driver) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if x, ok := h.inner.(driverapi.SQLQuerier); ok {
		return x.Exec(ctx, query, args...)
	}
	return nil, orm.ErrNotImplemented
}

func (h *driver) Query(ctx context.Context, query string, args ...any) (driverapi.Rows, error) {
	if x, ok := h.inner.(driverapi.SQLQuerier); ok {
		return x.Query(ctx, query, args...)
	}
	return nil, orm.ErrNotImplemented
}

func (h *driver) QueryRow(ctx context.Context, query string, args ...any) driverapi.Row {
	if x, ok := h.inner.(driverapi.SQLQuerier); ok {
		return x.QueryRow(ctx, query, args...)
	}
	return errRow{err: orm.ErrNotImplemented}
}

func (h *driver) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if x, ok := h.inner.(driverapi.Transactor); ok {
		return x.RunInTx(ctx, fn)
	}
	return orm.ErrNotImplemented
}

func (h *driver) Ping(ctx context.Context) error {
	if x, ok := h.inner.(driverapi.Pinger); ok {
		return x.Ping(ctx)
	}
	return orm.ErrNotImplemented
}

type errRow struct{ err error }

func (e errRow) Scan(dest ...any) error { return e.err }

var _ driverapi.Driver = (*driver)(nil)
var _ driverapi.SQLQuerier = (*driver)(nil)
