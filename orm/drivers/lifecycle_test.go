package drivers

import (
	"context"
	"errors"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestLifecycle_SaveWithoutInitReturnsNotInit(t *testing.T) {
	_ = Close(context.Background())
	err := DefaultDbDriver.Save(context.Background(), wrapperspb.String("x"))
	if !errors.Is(err, orm.ErrDbDriverNotInit) {
		t.Fatalf("Save() = %v, want ErrDbDriverNotInit", err)
	}
}

func TestLifecycle_TryInitNopThenClose(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	if !IsInitialized() {
		t.Fatal("expected initialized")
	}
	if CurrentDriverType() != DriverTypeNop {
		t.Fatalf("driver type = %q", CurrentDriverType())
	}
	if err := DefaultDbDriver.Save(context.Background(), wrapperspb.String("ok")); err != nil {
		t.Fatal(err)
	}
	if err := Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if IsInitialized() {
		t.Fatal("expected not initialized after Close")
	}
	err := DefaultDbDriver.Get(context.Background(), wrapperspb.String("x"))
	if !errors.Is(err, orm.ErrDbDriverNotInit) {
		t.Fatalf("after Close Get() = %v", err)
	}
}

func TestLifecycle_TryInitNilConf(t *testing.T) {
	_ = Close(context.Background())
	err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestTable())
	if !errors.Is(err, orm.ErrInvalidDriverOptions) {
		t.Fatalf("err = %v", err)
	}
}

func TestLifecycle_TryInitEmptyTables(t *testing.T) {
	_ = Close(context.Background())
	err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf())
	if !errors.Is(err, orm.ErrInvalidDriverOptions) {
		t.Fatalf("err = %v", err)
	}
}

func withTestConf() DriverOption {
	return WithConfig(&orm.Conf{Driver: string(DriverTypeNop)})
}

func withTestTable() DriverOption {
	return WithTables([]proto.Message{wrapperspb.String("")})
}

func TestLifecycle_DoubleTryInitNop(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	if !IsInitialized() {
		t.Fatal("expected initialized after double TryInit")
	}
	_ = Close(context.Background())
}

func TestTryInit_ConcurrentCalls(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	const n = 10
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			errCh <- TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable())
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("TryInit concurrent: %v", err)
		}
	}
	if !IsInitialized() {
		t.Fatal("expected initialized after concurrent TryInit")
	}
}

func TestPing_WithoutInit_ReturnsNotInit(t *testing.T) {
	_ = Close(context.Background())
	err := Ping(context.Background())
	if !errors.Is(err, orm.ErrDbDriverNotInit) {
		t.Fatalf("Ping() = %v", err)
	}
}

func TestIsInitialized_AndCurrentDriverType(t *testing.T) {
	_ = Close(context.Background())
	if IsInitialized() {
		t.Fatal("expected false before init")
	}
	if CurrentDriverType() != "" {
		t.Fatalf("type = %q, want empty", CurrentDriverType())
	}
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	if !IsInitialized() || CurrentDriverType() != DriverTypeNop {
		t.Fatalf("initialized=%v type=%q", IsInitialized(), CurrentDriverType())
	}
	_ = Close(context.Background())
}
