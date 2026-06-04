package logic

import (
	"context"
	"fmt"
	"gs/pbtest"
	"testing"

	"github.com/wxdqing/go-orm/orm/drivers"
	logger "git.wxdqing.com/sprout/logger.git"
)

func beforePgsql() {
	logger.Init()
	if err := UsePgsqlDriver(); err != nil {
		panic(err)
	}
}

func TestPgsqlSaveAndGet(t *testing.T) {
	beforePgsql()
	value := pbtest.Player{
		Id:   1001,
		Name: "pgsql_name_1",
	}
	if err := drivers.DefaultDbDriver.Save(context.Background(), &value); err != nil {
		t.Fatal(err)
	}
	got := &pbtest.Player{Id: 1001}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "pgsql_name_1" {
		t.Fatalf("name = %q, want pgsql_name_1", got.Name)
	}
}

func TestPgsqlFindByIndex(t *testing.T) {
	beforePgsql()
	for i := 0; i < 3; i++ {
		value := pbtest.Player{
			Id:   int64(2000 + i),
			Name: fmt.Sprintf("pgsql_list_%d", i),
		}
		if err := drivers.DefaultDbDriver.Save(context.Background(), &value); err != nil {
			t.Fatal(err)
		}
	}
	result, err := drivers.DefaultDbDriver.Find(context.Background(), &pbtest.Player{Name: "pgsql_list_1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one record")
	}
}
