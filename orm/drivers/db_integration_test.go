//go:build db

package drivers

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestLifecycle_DoubleTryInitMysqlConn(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	conf := testMySQLConf()
	tables := []proto.Message{wrapperspb.String("")}
	opts := []DriverOption{WithDriverType(DriverTypeMySQL), WithConfig(conf), WithTables(tables)}

	if err := TryInit(context.Background(), opts...); err != nil {
		t.Fatal(err)
	}
	open1 := gormOpenConns(t)
	if err := TryInit(context.Background(), opts...); err != nil {
		t.Fatal(err)
	}
	open2 := gormOpenConns(t)
	max := conf.Mysql.MaxOpen
	if max <= 0 {
		max = 10
	}
	if open2 > open1+max*2 {
		t.Fatalf("open connections grew too much: open1=%d open2=%d maxOpen=%d", open1, open2, max)
	}
}

func TestPing_AfterTryInit_MySQL(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	if err := TryInit(context.Background(),
		WithDriverType(DriverTypeMySQL),
		WithConfig(testMySQLConf()),
		WithTables([]proto.Message{wrapperspb.String("")}),
	); err != nil {
		t.Fatal(err)
	}
	if err := Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPing_AfterTryInit_Pgsql(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	if err := TryInit(context.Background(),
		WithDriverType(DriverTypePostgresSQL),
		WithConfig(testPgsqlConf()),
		WithTables([]proto.Message{wrapperspb.String("")}),
	); err != nil {
		t.Fatal(err)
	}
	if err := Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTryInit_RedisConnects(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	if err := TryInit(context.Background(),
		WithDriverType(DriverTypeRedis),
		WithConfig(testRedisConf()),
		withTestTable(),
	); err != nil {
		t.Fatal(err)
	}
}

func TestTryInit_MongoConnects(t *testing.T) {
	_ = Close(context.Background())
	defer Close(context.Background())

	if err := TryInit(context.Background(),
		WithDriverType(DriverTypeMongoDB),
		WithConfig(testMongoConf()),
		withTestTable(),
	); err != nil {
		t.Fatal(err)
	}
}

func gormOpenConns(t *testing.T) int {
	t.Helper()
	db := ToGorm()
	if db == nil {
		t.Fatal("ToGorm nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	return sqlDB.Stats().OpenConnections
}
