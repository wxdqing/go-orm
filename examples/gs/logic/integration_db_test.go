//go:build db

package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
	logger "git.wxdqing.com/sprout/logger.git"
)

func init() {
	logger.Init()
}

func integrationPlayerID(t *testing.T) int64 {
	t.Helper()
	return 900_000_000 + (time.Now().UnixNano() % 1_000_000)
}

func testVersionedPlayerCRUD(t *testing.T, init func() error) {
	t.Helper()
	_ = drivers.Close(context.Background())
	t.Cleanup(func() { _ = drivers.Close(context.Background()) })

	if err := init(); err != nil {
		t.Fatal(err)
	}

	id := integrationPlayerID(t)
	want := &pbtest.VersionedPlayer{
		Id:      id,
		Name:    "integration_crud",
		Level:   7,
		Avatar:  "av",
		Version: 3,
	}
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.VersionedPlayer{Id: id})
	})

	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := &pbtest.VersionedPlayer{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != want.Name || got.Level != want.Level || got.Avatar != want.Avatar || got.Version != want.Version {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if err := drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.VersionedPlayer{Id: id}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	err := drivers.DefaultDbDriver.Get(context.Background(), &pbtest.VersionedPlayer{Id: id})
	if !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatalf("after Delete Get: %v, want ErrRecordNotFound", err)
	}
}

func TestIntegration_MySQL_VersionedPlayer_CRUD(t *testing.T) {
	testVersionedPlayerCRUD(t, UseMysqlDriver)
}

func TestIntegration_Pgsql_VersionedPlayer_CRUD(t *testing.T) {
	testVersionedPlayerCRUD(t, UsePgsqlDriver)
}

func TestIntegration_Redis_VersionedPlayer_CRUD(t *testing.T) {
	testVersionedPlayerCRUD(t, UseRedisDriver)
}

func TestIntegration_Mongo_VersionedPlayer_CRUD(t *testing.T) {
	testVersionedPlayerCRUD(t, UseMongoDriver)
}
