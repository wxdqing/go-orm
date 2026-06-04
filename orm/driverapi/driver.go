package driverapi

import (
	"context"

	"github.com/wxdqing/go-orm/orm"
	"google.golang.org/protobuf/proto"
)

// Type 驱动类型标识（与 string 等价，便于写入 orm.Conf.Driver）。
type Type = string

const (
	TypeNop         Type = "nop"
	TypeMySQL       Type = "mysql"
	TypeRedis       Type = "redis"
	TypeTcaplusDB   Type = "tcaplus"
	TypeMongoDB     Type = "mongo"
	TypePostgresSQL Type = "pgsql"
)

// Options TryInit / InitDB 使用的配置（由 drivers 包组装）。
type Options struct {
	Type   Type
	Conf   *orm.Conf
	Tables []proto.Message
}

// Driver 数据驱动接口。
type Driver interface {
	InitDB(ctx context.Context, o *Options) error
	CloseDB(ctx context.Context) error
	Save(ctx context.Context, tb proto.Message) error
	Get(ctx context.Context, tb proto.Message) error
	Find(ctx context.Context, cond proto.Message) ([]proto.Message, error)
	Delete(ctx context.Context, tb proto.Message) error
}
