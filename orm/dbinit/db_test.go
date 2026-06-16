package dbinit

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/wxdqing/go-orm/orm"
)

func TestOpen_UnsupportedType(t *testing.T) {
	_, err := Open(context.Background(), "unknown", &orm.Conf{})
	if !errors.Is(err, orm.ErrInvalidDriverOptions) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpen_MissingConf(t *testing.T) {
	_, err := Open(context.Background(), TypeRedis, nil)
	if !errors.Is(err, orm.ErrInvalidDriverOptions) {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenRedis_Miniredis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	conf := &orm.Conf{Redis: orm.RedisConf{Host: mr.Addr()}}
	db, err := OpenRedis(context.Background(), conf)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(context.Background())

	if db.Redis() == nil {
		t.Fatal("expected redis client")
	}
	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() err = %v", err)
	}
}

func TestGroup_Close(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	db, err := OpenRedis(context.Background(), &orm.Conf{Redis: orm.RedisConf{Host: mr.Addr()}})
	if err != nil {
		t.Fatal(err)
	}
	var g Group
	g.Add(db)
	if err := g.Close(context.Background()); err != nil {
		t.Fatalf("Close() err = %v", err)
	}
	if err := db.Ping(context.Background()); err == nil {
		t.Fatal("expected ping error after close")
	}
}
