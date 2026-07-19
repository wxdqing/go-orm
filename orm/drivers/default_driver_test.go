package drivers

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type sqlStubDriver struct {
	execQuery string
	closed    atomic.Bool
	saveGate  chan struct{}
	saveHold  chan struct{}
	db        *sql.DB
}

func (d *sqlStubDriver) InitDB(context.Context, *driverapi.Options) error { return nil }
func (d *sqlStubDriver) CloseDB(context.Context) error {
	d.closed.Store(true)
	return nil
}
func (d *sqlStubDriver) Save(context.Context, proto.Message) error {
	if d.saveGate != nil {
		close(d.saveGate)
	}
	if d.saveHold != nil {
		<-d.saveHold
	}
	if d.closed.Load() {
		return errors.New("save after close")
	}
	return nil
}
func (d *sqlStubDriver) Get(context.Context, proto.Message) error  { return nil }
func (d *sqlStubDriver) Find(context.Context, proto.Message) ([]proto.Message, error) {
	return nil, nil
}
func (d *sqlStubDriver) Delete(context.Context, proto.Message) error { return nil }
func (d *sqlStubDriver) Count(context.Context, proto.Message) (int64, error) {
	return 0, nil
}
func (d *sqlStubDriver) RunInTx(context.Context, func(context.Context) error) error {
	return nil
}
func (d *sqlStubDriver) Ping(context.Context) error { return nil }
func (d *sqlStubDriver) Exec(_ context.Context, query string, _ ...any) (sql.Result, error) {
	d.execQuery = query
	return stubResult{}, nil
}
func (d *sqlStubDriver) Query(ctx context.Context, query string, args ...any) (driverapi.Rows, error) {
	if d.db == nil {
		return nil, orm.ErrNotImplemented
	}
	return d.db.QueryContext(ctx, query, args...)
}
func (d *sqlStubDriver) QueryRow(ctx context.Context, query string, args ...any) driverapi.Row {
	if d.db == nil {
		return errScanRow{err: orm.ErrNotImplemented}
	}
	return d.db.QueryRowContext(ctx, query, args...)
}

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) { return 1, nil }
func (stubResult) RowsAffected() (int64, error) { return 1, nil }

type errScanRow struct{ err error }

func (e errScanRow) Scan(...any) error { return e.err }

func TestDefaultDriverForwardsSQLQuerier(t *testing.T) {
	_ = Close(context.Background())
	stub := &sqlStubDriver{}
	defaultDriver.set(stub)
	t.Cleanup(func() { defaultDriver.set(lifecycle.closed) })

	querier, ok := DefaultDbDriver.(SQLQuerier)
	if !ok {
		t.Fatal("DefaultDbDriver does not implement SQLQuerier")
	}
	if _, err := querier.Exec(context.Background(), "SELECT 1"); err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if stub.execQuery != "SELECT 1" {
		t.Fatalf("Exec() query = %q, want SELECT 1", stub.execQuery)
	}
}

func TestCloseWaitsForInFlightSave(t *testing.T) {
	proxy := &driverProxy{}
	started := make(chan struct{})
	hold := make(chan struct{})
	stub := &sqlStubDriver{saveGate: started, saveHold: hold}
	proxy.set(stub)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = proxy.Save(context.Background(), wrapperspb.String("x"))
	}()
	<-started

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- proxy.closeAndSet(context.Background(), lifecycle.closed)
	}()

	select {
	case err := <-closeDone:
		t.Fatalf("closeAndSet returned while Save in flight: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(hold)
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("closeAndSet() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("closeAndSet did not finish after Save completed")
	}
	wg.Wait()
	if !stub.closed.Load() {
		t.Fatal("underlying CloseDB was not called")
	}
}

func openStubDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestQueryPinReleasedAfterRowsClose(t *testing.T) {
	proxy := &driverProxy{}
	stub := &sqlStubDriver{db: openStubDB(t)}
	proxy.set(stub)

	rows, err := proxy.Query(context.Background(), "select 1")
	if err != nil {
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := proxy.closeAndSet(context.Background(), lifecycle.closed); err != nil {
		t.Fatalf("closeAndSet() error = %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("closeAndSet waited after rows were closed: %s", time.Since(start))
	}
	if !stub.closed.Load() {
		t.Fatal("underlying CloseDB was not called")
	}
}

func TestQueryRowPinReleasedAfterScan(t *testing.T) {
	proxy := &driverProxy{}
	stub := &sqlStubDriver{db: openStubDB(t)}
	proxy.set(stub)

	row := proxy.QueryRow(context.Background(), "select 1")
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := proxy.closeAndSet(context.Background(), lifecycle.closed); err != nil {
		t.Fatalf("closeAndSet() error = %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("closeAndSet waited after Scan: %s", time.Since(start))
	}
}

func TestCloseTimeoutReturnsError(t *testing.T) {
	prevTimeout := closeWaitTimeout
	closeWaitTimeout = 30 * time.Millisecond
	t.Cleanup(func() { closeWaitTimeout = prevTimeout })

	proxy := &driverProxy{}
	started := make(chan struct{})
	hold := make(chan struct{})
	stub := &sqlStubDriver{saveGate: started, saveHold: hold}
	proxy.set(stub)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = proxy.Save(context.Background(), wrapperspb.String("x"))
	}()
	<-started

	err := proxy.closeAndSet(context.Background(), lifecycle.closed)
	if !errors.Is(err, orm.ErrDbDriverCloseTimeout) {
		t.Fatalf("closeAndSet() error = %v, want ErrDbDriverCloseTimeout", err)
	}
	if !stub.closed.Load() {
		t.Fatal("CloseDB should still run after timeout")
	}
	close(hold)
	wg.Wait()
}

func TestCloseReleasesAbandonedQueryPin(t *testing.T) {
	proxy := &driverProxy{}
	stub := &sqlStubDriver{db: openStubDB(t)}
	proxy.set(stub)

	rows, err := proxy.Query(context.Background(), "select 1")
	if err != nil {
		t.Fatal(err)
	}
	_ = rows // intentionally not closed

	start := time.Now()
	if err := proxy.closeAndSet(context.Background(), lifecycle.closed); err != nil {
		t.Fatalf("closeAndSet() error = %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("closeAndSet waited too long for abandoned Query pin: %s", time.Since(start))
	}
	if !stub.closed.Load() {
		t.Fatal("underlying CloseDB was not called")
	}
}

func TestCloseReleasesAbandonedQueryRowPin(t *testing.T) {
	proxy := &driverProxy{}
	stub := &sqlStubDriver{db: openStubDB(t)}
	proxy.set(stub)

	_ = proxy.QueryRow(context.Background(), "select 1") // intentionally not scanned

	start := time.Now()
	if err := proxy.closeAndSet(context.Background(), lifecycle.closed); err != nil {
		t.Fatalf("closeAndSet() error = %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("closeAndSet waited too long for abandoned QueryRow pin: %s", time.Since(start))
	}
}

func TestCloseInsideRunInTxDoesNotDeadlock(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- DefaultDbDriver.RunInTx(context.Background(), func(context.Context) error {
			return Close(context.Background())
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunInTx/Close error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close inside RunInTx deadlocked")
	}
	if IsInitialized() {
		t.Fatal("expected lifecycle closed")
	}
}

func TestCloseDuringSlowInitDoesNotBlock(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	started := make(chan struct{})
	hold := make(chan struct{})
	prev := slowInitHook
	slowInitHook = func() {
		close(started)
		<-hold
	}
	t.Cleanup(func() { slowInitHook = prev })

	initDone := make(chan error, 1)
	go func() {
		initDone <- TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable())
	}()
	<-started

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- Close(context.Background())
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Close blocked while InitDB was in progress")
	}

	close(hold)
	if err := <-initDone; err != nil {
		t.Fatalf("TryInit() error = %v", err)
	}
}

func TestCloseDetachesBeforeWaitingForInFlight(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	hold := make(chan struct{})
	txDone := make(chan error, 1)
	go func() {
		txDone <- DefaultDbDriver.RunInTx(context.Background(), func(context.Context) error {
			close(started)
			<-hold
			return nil
		})
	}()
	<-started

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- Close(context.Background())
	}()

	detached := false
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !IsInitialized() {
			detached = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !detached {
		t.Fatal("Close held lifecycle initialized while waiting for in-flight RunInTx")
	}

	close(hold)
	if err := <-txDone; err != nil {
		t.Fatalf("RunInTx() error = %v", err)
	}
	if err := <-closeDone; err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
