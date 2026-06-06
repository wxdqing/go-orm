package codec

import (
	"context"

	"github.com/wxdqing/go-orm/orm"
	"google.golang.org/protobuf/proto"
)

// ensureCtx 调用方传入 nil 时使用 Background。
func EnsureCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// protoCtx 与 protoc-gen-go-orm 生成的 EncodeFromContext Context 兼容。
type protoCtx interface {
	Value(key any) any
}

type recordEncoder interface {
	EncodeFromContext(ctx protoCtx, value proto.Message) error
}

type recordDecoder interface {
	DecodeToContext(ctx protoCtx, value proto.Message) error
}

func EncodeRecord(ctx context.Context, dbObj proto.Message, value proto.Message) error {
	if enc, ok := dbObj.(recordEncoder); ok {
		return enc.EncodeFromContext(EnsureCtx(ctx), value)
	}
	if enc, ok := dbObj.(orm.PersistEncoder); ok {
		return enc.EncodeFrom(value)
	}
	return orm.ErrNotTableRecord
}

func DecodeRecord(ctx context.Context, dbObj proto.Message, value proto.Message) error {
	// 写回业务 proto 前先 Reset：PK/条件阶段仍用调用方传入对象，仅结果赋值时清空再解码。
	if value != nil {
		proto.Reset(value)
	}
	if dec, ok := dbObj.(recordDecoder); ok {
		return dec.DecodeToContext(EnsureCtx(ctx), value)
	}
	if dec, ok := dbObj.(orm.PersistDecoder); ok {
		return dec.DecodeTo(value)
	}
	return orm.ErrNotTableRecord
}
