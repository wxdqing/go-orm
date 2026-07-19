package drivers

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
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

func TestDefaultDriverConcurrentUseAndReinit(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	defer Close(context.Background())

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 500; j++ {
				_ = DefaultDbDriver.Save(context.Background(), wrapperspb.String("value"))
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 50; i++ {
			_ = Close(context.Background())
			_ = TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable())
		}
	}()
	close(start)
	wg.Wait()
}

func TestRunInTxAllowsNestedProxyCalls(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	defer Close(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- DefaultDbDriver.RunInTx(context.Background(), func(txCtx context.Context) error {
			if err := DefaultDbDriver.Ping(txCtx); err != nil {
				return err
			}
			if err := DefaultDbDriver.Save(txCtx, wrapperspb.String("in-tx")); err != nil {
				return err
			}
			return DefaultDbDriver.RunInTx(txCtx, func(context.Context) error {
				return DefaultDbDriver.Ping(context.Background())
			})
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunInTx() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunInTx nested proxy calls deadlocked")
	}
}

func TestPingUsesActiveRedisDriver(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	defer Close(context.Background())

	err = TryInit(context.Background(),
		WithDriverType(DriverTypeRedis),
		WithConfig(&orm.Conf{Driver: string(DriverTypeRedis), Redis: orm.RedisConf{Host: mr.Addr()}}),
		withTestTable(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestBuildDriverOptionsReturnsConfigMapError(t *testing.T) {
	if os.Getenv("GO_ORM_CONFIG_ERROR_CHILD") == "1" {
		_, err := buildDriverOptions(
			WithConfigMap(map[string]any{"db": map[string]any{
				"mysql": map[string]any{"max_open": []string{"invalid"}},
			}}),
			withTestTable(),
		)
		if err == nil {
			t.Fatal("buildDriverOptions() error = nil")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBuildDriverOptionsReturnsConfigMapError$")
	cmd.Env = append(os.Environ(), "GO_ORM_CONFIG_ERROR_CHILD=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("config error terminated process: %v\n%s", err, out)
	}
}

func TestBuildDriverOptionsRejectsNilConfig(t *testing.T) {
	_, err := buildDriverOptions(WithConfig(nil), withTestTable())
	if !errors.Is(err, orm.ErrInvalidDriverOptions) {
		t.Fatalf("buildDriverOptions() error = %v, want ErrInvalidDriverOptions", err)
	}
}

func TestBuildDriverOptionsDoesNotPublishMetadata(t *testing.T) {
	_ = Close(context.Background())
	if _, err := buildDriverOptions(withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	if DbTableNameMapping != nil || ValueNameMapping != nil {
		t.Fatal("buildDriverOptions() published global metadata before driver initialization")
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

func TestDefaultDriverCloseDBUpdatesLifecycle(t *testing.T) {
	_ = Close(context.Background())
	if err := TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable()); err != nil {
		t.Fatal(err)
	}
	if err := DefaultDbDriver.CloseDB(context.Background()); err != nil {
		t.Fatal(err)
	}
	if IsInitialized() {
		t.Fatal("lifecycle remains initialized after DefaultDbDriver.CloseDB")
	}
	if err := DefaultDbDriver.Save(context.Background(), wrapperspb.String("x")); !errors.Is(err, orm.ErrDbDriverNotInit) {
		t.Fatalf("Save() error = %v, want ErrDbDriverNotInit", err)
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

func TestTryInitRechecksReadyAfterCloseWait(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

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

	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			errCh <- TryInit(context.Background(), WithDriverType(DriverTypeNop), withTestConf(), withTestTable())
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(hold)
	if err := <-txDone; err != nil {
		t.Fatalf("RunInTx() error = %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("TryInit() error = %v", err)
		}
	}
	if !IsInitialized() || CurrentDriverType() != DriverTypeNop {
		t.Fatalf("initialized=%v type=%q", IsInitialized(), CurrentDriverType())
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
