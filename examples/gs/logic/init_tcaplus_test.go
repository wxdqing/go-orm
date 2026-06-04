//go:build tcaplus

// Tcaplus 集成测试默认不跑；需要时：go test -tags=tcaplus ./...

package logic

import (
	"context"
	"errors"
	"gs/pbtest"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	logger "git.wxdqing.com/sprout/logger.git"
)

func tcaplusBeforeAll(t *testing.T) {
	t.Helper()
	logger.Init()
	if err := UseTcaplusDriver(); err != nil {
		t.Fatal(err)
	}
}

func TestTcaSaveWithVersion(t *testing.T) {
	tcaplusBeforeAll(t)
	value := pbtest.VersionedPlayer{
		Id:   1,
		Name: "name_1",
	}
	err := drivers.DefaultDbDriver.Get(context.Background(), &value)
	if err != nil {
		if errors.Is(err, orm.ErrRecordNotFound) {
			drivers.DefaultDbDriver.Save(context.Background(), &value)
		} else {
			panic(err)
		}
	}
	value.Name = "name_2"
	err = drivers.DefaultDbDriver.Save(context.Background(), &value)
	if err != nil {
		panic(err)
	}
	logger.Info("init mysql save with cond player success")
}
