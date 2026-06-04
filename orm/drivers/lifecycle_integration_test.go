//go:build db

package drivers

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// T-LC-004：Close 后 primary *sql.DB 不可 Ping。
func TestLifecycle_GormCloseClosesPrimary(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	if err := TryInit(context.Background(),
		WithDriverType(DriverTypeMySQL),
		WithConfig(testMySQLConf()),
		WithTables([]proto.Message{wrapperspb.String("")}),
	); err != nil {
		t.Fatal(err)
	}
	gdb := ToGorm()
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.PingContext(context.Background()); err == nil {
		t.Fatal("expected ping failure after Close")
	} else if !errors.Is(err, sql.ErrConnDone) {
		t.Logf("ping after close: %v", err)
	}
}

// T-LC-007：连接池参数应用到 primary。
func TestLifecycle_PoolMaxOpenAppliedToPrimary(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	conf := testMySQLConf()
	conf.Mysql.MaxOpen = 3
	conf.Mysql.MaxIdle = 2

	if err := TryInit(context.Background(),
		WithDriverType(DriverTypeMySQL),
		WithConfig(conf),
		WithTables([]proto.Message{wrapperspb.String("")}),
	); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := ToGorm().DB()
	if err != nil {
		t.Fatal(err)
	}
	stats := sqlDB.Stats()
	if stats.MaxOpenConnections != 3 {
		t.Fatalf("MaxOpenConnections = %d, want 3", stats.MaxOpenConnections)
	}
}
