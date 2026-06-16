package driverapi

import (
	"context"
	"database/sql"

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

// Row 单行查询结果；*sql.Row 与错误占位均实现此接口。
type Row interface {
	Scan(dest ...any) error
}

// Driver 数据驱动接口（proto CRUD + 事务 + 连通性）。
type Driver interface {
	InitDB(ctx context.Context, o *Options) error
	CloseDB(ctx context.Context) error
	Save(ctx context.Context, tb proto.Message) error
	Get(ctx context.Context, tb proto.Message) error
	Find(ctx context.Context, cond proto.Message) ([]proto.Message, error)
	Delete(ctx context.Context, tb proto.Message) error
	Count(ctx context.Context, cond proto.Message) (int64, error)
	// RunInTx 在事务中执行 fn；fn 收到的 ctx 可传给 Save/Find/Delete 等。
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
	Ping(ctx context.Context) error
}

// SQLQuerier 原始 SQL 桥接（仅 SQL 驱动可选实现；业务 Store 不应依赖）。
type SQLQuerier interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) Row
}

// SQLDriver 同时具备 proto CRUD 与 raw SQL 能力（mysql/pgsql）。
type SQLDriver interface {
	Driver
	SQLQuerier
}
