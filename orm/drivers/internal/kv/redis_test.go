package kv

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"google.golang.org/protobuf/types/known/emptypb"
)

// T-KV-001：未注册表 Save 返回 ErrNotTableRecord
func TestRedis_Save_NotTableRecord(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	r := NewRedis().(*Redis)
	ctx := context.Background()
	if err := r.InitDB(ctx, &driverapi.Options{Conf: &orm.Conf{Redis: orm.RedisConf{Host: mr.Addr()}}}); err != nil {
		t.Fatal(err)
	}
	defer r.CloseDB(ctx)

	err = r.Save(ctx, &emptypb.Empty{})
	if err != orm.ErrNotTableRecord {
		t.Fatalf("Save() err = %v, want ErrNotTableRecord", err)
	}
}

// T-KV-002：未初始化返回 ErrDbDriverNotInit
func TestRedis_Save_NotInit(t *testing.T) {
	r := NewRedis().(*Redis)
	err := r.Save(context.Background(), &emptypb.Empty{})
	if err != orm.ErrDbDriverNotInit {
		t.Fatalf("Save() err = %v, want ErrDbDriverNotInit", err)
	}
}
