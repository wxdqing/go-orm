package logic

import (
	"context"
	"errors"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
	"gs/pbtest/metadata"
	logger "git.wxdqing.com/sprout/logger.git"
)

func beforeAllWithHandlers(t *testing.T) {
	t.Helper()
	logger.Init()
	versionedPlayerCustomSaveCount.Store(0)
	if err := drivers.TryInit(context.Background(),
		drivers.WithTables(metadata.GetAllTables(drivers.DriverTypeMySQL)),
		drivers.WithConfig(&orm.Conf{
			Driver: drivers.DriverTypeMySQL,
			Mysql: orm.MysqlConf{
				Addr:     "localhost:3306",
				Name:     "game",
				User:     "root",
				Password: "root123",
				Startup:  orm.DefaultGormStartup("mysql"),
			},
		}),
		drivers.WithHandlerRegistry(ExampleHandlerRegistry()),
	); err != nil {
		t.Fatal(err)
	}
}

func TestPlayerSaveBeforeHookRejectsEmptyName(t *testing.T) {
	beforeAllWithHandlers(t)

	err := drivers.DefaultDbDriver.Save(context.Background(), &pbtest.Player{Id: 9001, Name: ""})
	if err == nil {
		t.Fatal("expected before hook to reject empty name")
	}
}

func TestPlayerSaveBeforeHookAllowsValidName(t *testing.T) {
	beforeAllWithHandlers(t)

	p := &pbtest.Player{Id: 9002, Name: "handler_ok"}
	if err := drivers.DefaultDbDriver.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if err := drivers.DefaultDbDriver.Get(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	_ = drivers.DefaultDbDriver.Delete(context.Background(), p)
}

func TestVersionedPlayerCustomSaveBypassesGorm(t *testing.T) {
	beforeAllWithHandlers(t)

	vp := &pbtest.VersionedPlayer{Id: 9100, Name: "custom_save"}
	if err := drivers.DefaultDbDriver.Save(context.Background(), vp); err != nil {
		t.Fatal(err)
	}
	if VersionedPlayerCustomSaveCount() != 1 {
		t.Fatalf("custom save count = %d, want 1", VersionedPlayerCustomSaveCount())
	}
	// 自定义 Save 未走 GORM，库中应无记录（用新对象 Get，避免 vp 上残留字段误判）
	loaded := &pbtest.VersionedPlayer{Id: 9100}
	err := drivers.DefaultDbDriver.Get(context.Background(), loaded)
	if err == nil && loaded.GetName() != "" {
		t.Fatalf("expected no row in db after custom save, got %+v", loaded)
	}
	if err != nil && !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatal(err)
	}
}
