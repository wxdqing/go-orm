package logic

import (
	"context"
	"errors"
	"fmt"
	"gs/pbtest"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	logger "gitee.com/wxdqing/logger.git"
)

func TestInit(t *testing.T) {
	beforeAll()
}

func beforeAll() {
	logger.Init()
	if err := UseMysqlDriver(); err != nil {
		panic(err)
	}
}

func TestSaveWithVersion(t *testing.T) {
	beforeAll()
	value := pbtest.VersionedPlayer{
		Id:   1,
		Name: "name_1",
	}
	err := drivers.DefaultDbDriver.Get(context.Background(), &value)
	if errors.Is(err, orm.ErrRecordNotFound) {
		if err = drivers.DefaultDbDriver.Save(context.Background(), &value); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	value.Name = "name_2"
	if err = drivers.DefaultDbDriver.Save(context.Background(), &value); err != nil {
		t.Fatal(err)
	}
	logger.Info("init mysql save with cond player success")
}

func TestMysqlFind(t *testing.T) {
	beforeAll()
	save(t, 10)
	result, err := drivers.DefaultDbDriver.Find(context.Background(), &pbtest.Player{Name: "name_2"})
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("init mysql find player result is [%v]", result)
}

func TestMysqlFindByIndex(t *testing.T) {
	beforeAll()
	save(t, 10)
	result, err := drivers.DefaultDbDriver.Find(context.Background(), &pbtest.Player{Name: "name_2"})
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("init mysql get list player result is [%v]", result)
}

func TestSaveMysql(t *testing.T) {
	beforeAll()
	save(t, 10)
	logger.Info("init mysql save player success")
	result := &pbtest.Player{Id: 1, Name: "abc"}
	err := drivers.DefaultDbDriver.Get(context.Background(), result)
	//result, err := drivers.DefaultDbDriver.GetAll(&pbtest.Player{})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
	//logger.Infof("init mysql get all player result is [%d] [%v]", len(result), result)
}

func save(t *testing.T, num int) {
	for i := 0; i < num; i++ {
		value := pbtest.Player{
			Id:   int64(i + 1),
			Name: fmt.Sprintf("name_%d", i+1),
		}
		if err := drivers.DefaultDbDriver.Save(context.Background(), &value); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetOne(t *testing.T) {
	beforeAll()
	save(t, 1)
	tb := &pbtest.Player{Id: 1}
	err := drivers.DefaultDbDriver.Get(context.Background(), tb)
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("init mysql get one player result is [%v]", tb)
}

func TestDeleteMysql(t *testing.T) {
	beforeAll()
	save(t, 10)
	tb := &pbtest.Player{Id: 1}
	err := drivers.DefaultDbDriver.Get(context.Background(), tb)
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("init mysql get one player result is [%v]", tb)

	// delete
	err = drivers.DefaultDbDriver.Delete(context.Background(), tb)
	if err != nil {
		t.Fatal(err)
	}

	//result, err := drivers.DefaultDbDriver.GetAll(&pbtest.Player{})
	//if err != nil {
	//	panic(err)
	//}
	//logger.Infof("init mysql get all player result is [%d] [%v]", len(result), result)
}
