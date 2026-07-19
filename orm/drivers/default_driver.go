package drivers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"google.golang.org/protobuf/proto"
)

type transactionDriverKey struct{}

// Tunables for tests; production keeps the defaults.
var (
	closeWaitTimeout = 30 * time.Second
	closeWaitPoll    = time.Millisecond
)

// txDepthByG tracks nested RunInTx depth per goroutine so Close can skip
// waiting on the closing goroutine's own transaction pins.
var txDepthByG sync.Map // int64 goid -> *atomic.Int32

type pinHolder interface {
	forceRelease()
}

// activeDriver boxes a CoreDriver so atomic.Pointer can publish interface values.
// refs counts in-flight operations that pinned this generation; close waits until
// refs drops to the closing goroutine's RunInTx depth (usually 0).
type activeDriver struct {
	driver CoreDriver
	refs   atomic.Int32

	holdMu sync.Mutex
	holds  []pinHolder
}

type driverProxy struct {
	active atomic.Pointer[activeDriver]
}

func (p *driverProxy) set(active CoreDriver) {
	p.active.Store(&activeDriver{driver: active})
}

func (p *driverProxy) replace(next CoreDriver) *activeDriver {
	return p.active.Swap(&activeDriver{driver: next})
}

func (p *driverProxy) closeAndSet(ctx context.Context, next CoreDriver) error {
	return closePrevious(ctx, p.replace(next))
}

func closePrevious(ctx context.Context, prev *activeDriver) error {
	if prev == nil || prev.driver == nil {
		return nil
	}
	prev.forceReleaseHolds()
	waitErr := waitForRefs(prev)
	closeErr := prev.driver.CloseDB(ctx)
	if waitErr != nil {
		orm.GetLogger().Errorf("orm close: %v (in-flight=%d)", waitErr, prev.refs.Load())
		return errors.Join(waitErr, closeErr)
	}
	return closeErr
}

func waitForRefs(active *activeDriver) error {
	allow := currentTxDepth()
	deadline := time.Now().Add(closeWaitTimeout)
	for active.refs.Load() > allow {
		if time.Now().After(deadline) {
			return fmt.Errorf("%w: remaining=%d", orm.ErrDbDriverCloseTimeout, active.refs.Load())
		}
		time.Sleep(closeWaitPoll)
	}
	return nil
}

func enterTx() {
	id := currentGoID()
	v, _ := txDepthByG.LoadOrStore(id, new(atomic.Int32))
	v.(*atomic.Int32).Add(1)
}

func leaveTx() {
	id := currentGoID()
	if v, ok := txDepthByG.Load(id); ok {
		if v.(*atomic.Int32).Add(-1) == 0 {
			txDepthByG.Delete(id)
		}
	}
}

func currentTxDepth() int32 {
	if v, ok := txDepthByG.Load(currentGoID()); ok {
		return v.(*atomic.Int32).Load()
	}
	return 0
}

func (a *activeDriver) track(h pinHolder) {
	a.holdMu.Lock()
	a.holds = append(a.holds, h)
	a.holdMu.Unlock()
}

func (a *activeDriver) untrack(h pinHolder) {
	a.holdMu.Lock()
	defer a.holdMu.Unlock()
	for i, cur := range a.holds {
		if cur == h {
			a.holds = append(a.holds[:i], a.holds[i+1:]...)
			return
		}
	}
}

func (a *activeDriver) forceReleaseHolds() {
	a.holdMu.Lock()
	holds := a.holds
	a.holds = nil
	a.holdMu.Unlock()
	for _, h := range holds {
		h.forceRelease()
	}
}

func (p *driverProxy) current() CoreDriver {
	if active := p.active.Load(); active != nil {
		return active.driver
	}
	return nil
}

func (p *driverProxy) pin() *activeDriver {
	for {
		active := p.active.Load()
		if active == nil || active.driver == nil {
			fallback := &activeDriver{driver: lifecycle.closed}
			fallback.refs.Add(1)
			return fallback
		}
		active.refs.Add(1)
		if p.active.Load() == active {
			return active
		}
		active.refs.Add(-1)
	}
}

func (a *activeDriver) unpin() {
	if a != nil {
		a.refs.Add(-1)
	}
}

func currentGoID() int64 {
	buf := make([]byte, 128)
	n := runtime.Stack(buf, false)
	s := string(buf[:n])
	const prefix = "goroutine "
	if !strings.HasPrefix(s, prefix) {
		return -1
	}
	s = s[len(prefix):]
	if i := strings.IndexByte(s, ' '); i >= 0 {
		s = s[:i]
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return -1
	}
	return id
}

// acquire returns the driver for this call. Transaction-scoped drivers are not
// pinned; the surrounding RunInTx pin covers them. Otherwise the active
// generation is pinned until release is called.
func (p *driverProxy) acquire(ctx context.Context) (CoreDriver, func()) {
	if driver := transactionDriver(ctx); driver != nil {
		return driver, func() {}
	}
	active := p.pin()
	return active.driver, active.unpin
}

func (p *driverProxy) Unwrap() CoreDriver {
	return p.current()
}

func transactionDriver(ctx context.Context) CoreDriver {
	if ctx == nil {
		return nil
	}
	driver, _ := ctx.Value(transactionDriverKey{}).(CoreDriver)
	return driver
}

func (p *driverProxy) InitDB(ctx context.Context, o *driverapi.Options) error {
	active := p.pin()
	defer active.unpin()
	return active.driver.InitDB(ctx, o)
}

func (p *driverProxy) CloseDB(ctx context.Context) error {
	return Close(ctx)
}

func (p *driverProxy) Save(ctx context.Context, value proto.Message) error {
	driver, release := p.acquire(ctx)
	defer release()
	return driver.Save(ctx, value)
}

func (p *driverProxy) Get(ctx context.Context, value proto.Message) error {
	driver, release := p.acquire(ctx)
	defer release()
	return driver.Get(ctx, value)
}

func (p *driverProxy) Find(ctx context.Context, cond proto.Message) ([]proto.Message, error) {
	driver, release := p.acquire(ctx)
	defer release()
	if finder, ok := driver.(Finder); ok {
		return finder.Find(ctx, cond)
	}
	return nil, orm.ErrNotImplemented
}

func (p *driverProxy) Delete(ctx context.Context, value proto.Message) error {
	driver, release := p.acquire(ctx)
	defer release()
	return driver.Delete(ctx, value)
}

func (p *driverProxy) Count(ctx context.Context, cond proto.Message) (int64, error) {
	driver, release := p.acquire(ctx)
	defer release()
	if counter, ok := driver.(Counter); ok {
		return counter.Count(ctx, cond)
	}
	return 0, orm.ErrNotImplemented
}

func (p *driverProxy) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	enterTx()
	defer leaveTx()
	active := p.pin()
	defer active.unpin()
	transactor, ok := active.driver.(Transactor)
	if !ok {
		return orm.ErrNotImplemented
	}
	return transactor.RunInTx(ctx, func(txCtx context.Context) error {
		return fn(context.WithValue(txCtx, transactionDriverKey{}, active.driver))
	})
}

func (p *driverProxy) Ping(ctx context.Context) error {
	driver, release := p.acquire(ctx)
	defer release()
	if pinger, ok := driver.(Pinger); ok {
		return pinger.Ping(ctx)
	}
	return orm.ErrNotImplemented
}

func (p *driverProxy) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	driver, release := p.acquire(ctx)
	defer release()
	if querier, ok := driver.(SQLQuerier); ok {
		return querier.Exec(ctx, query, args...)
	}
	return nil, orm.ErrNotImplemented
}

func (p *driverProxy) Query(ctx context.Context, query string, args ...any) (driverapi.Rows, error) {
	if driver := transactionDriver(ctx); driver != nil {
		querier, ok := driver.(SQLQuerier)
		if !ok {
			return nil, orm.ErrNotImplemented
		}
		return querier.Query(ctx, query, args...)
	}
	active := p.pin()
	querier, ok := active.driver.(SQLQuerier)
	if !ok {
		active.unpin()
		return nil, orm.ErrNotImplemented
	}
	rows, err := querier.Query(ctx, query, args...)
	if err != nil || rows == nil {
		active.unpin()
		return rows, err
	}
	return wrapPinnedRows(rows, active), nil
}

func (p *driverProxy) QueryRow(ctx context.Context, query string, args ...any) driverapi.Row {
	if driver := transactionDriver(ctx); driver != nil {
		querier, ok := driver.(SQLQuerier)
		if !ok {
			return errRow{err: orm.ErrNotImplemented}
		}
		return querier.QueryRow(ctx, query, args...)
	}
	active := p.pin()
	querier, ok := active.driver.(SQLQuerier)
	if !ok {
		active.unpin()
		return errRow{err: orm.ErrNotImplemented}
	}
	return wrapPinnedRow(querier.QueryRow(ctx, query, args...), active)
}

type pinnedRows struct {
	driverapi.Rows
	active  *activeDriver
	once    sync.Once
	tracked bool
}

func wrapPinnedRows(rows driverapi.Rows, active *activeDriver) driverapi.Rows {
	r := &pinnedRows{Rows: rows, active: active, tracked: true}
	active.track(r)
	return r
}

func (r *pinnedRows) Close() error {
	err := r.Rows.Close()
	r.releasePin()
	return err
}

func (r *pinnedRows) forceRelease() {
	_ = r.Rows.Close()
	r.releasePin()
}

func (r *pinnedRows) releasePin() {
	r.once.Do(func() {
		if r.tracked {
			r.active.untrack(r)
		}
		r.active.unpin()
	})
}

type pinnedRow struct {
	row     driverapi.Row
	active  *activeDriver
	once    sync.Once
	tracked bool
}

func wrapPinnedRow(row driverapi.Row, active *activeDriver) driverapi.Row {
	r := &pinnedRow{row: row, active: active, tracked: true}
	active.track(r)
	return r
}

func (r *pinnedRow) Scan(dest ...any) error {
	defer r.releasePin()
	return r.row.Scan(dest...)
}

func (r *pinnedRow) forceRelease() {
	r.releasePin()
}

func (r *pinnedRow) releasePin() {
	r.once.Do(func() {
		if r.tracked {
			r.active.untrack(r)
		}
		r.active.unpin()
	})
}

type errRow struct{ err error }

func (e errRow) Scan(dest ...any) error { return e.err }

var _ Driver = (*driverProxy)(nil)
var _ SQLQuerier = (*driverProxy)(nil)
