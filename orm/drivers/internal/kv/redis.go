package kv

import (
	"context"
	"fmt"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type Redis struct {
	client *redis.Client
}

func NewRedis() driverapi.Driver {
	return &Redis{}
}

func (r *Redis) InitDB(ctx context.Context, o *driverapi.Options) error {
	c := o.Conf.Redis
	if c.Host == "" {
		return fmt.Errorf("%w: redis host is required", orm.ErrInvalidDriverOptions)
	}
	r.client = redis.NewClient(&redis.Options{
		Addr:     c.Host,
		Password: c.Password,
		DB:       c.Index,
	})
	if err := r.client.Ping(ctx).Err(); err != nil {
		_ = r.client.Close()
		r.client = nil
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (r *Redis) CloseDB(context.Context) error {
	if r.client == nil {
		return nil
	}
	err := r.client.Close()
	r.client = nil
	return err
}

func (r *Redis) Save(ctx context.Context, value proto.Message) error {
	if r.client == nil {
		return orm.ErrDbDriverNotInit
	}
	tm := meta.GetMetaByValue(value)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, value); err != nil {
		return err
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	key, err := recordKey(tableName(dbObj), pk)
	if err != nil {
		return err
	}
	payload, err := proto.Marshal(value)
	if err != nil {
		return err
	}
	if err := r.client.Set(ctx, key, payload, 0).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Redis) Get(ctx context.Context, value proto.Message) error {
	if r.client == nil {
		return orm.ErrDbDriverNotInit
	}
	tm := meta.GetMetaByValue(value)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, value); err != nil {
		return err
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	key, err := recordKey(tableName(dbObj), pk)
	if err != nil {
		return err
	}
	b, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return orm.ErrRecordNotFound
	}
	if err != nil {
		return err
	}
	proto.Reset(value)
	if err := proto.Unmarshal(b, value); err != nil {
		return err
	}
	return nil
}

func (r *Redis) Find(context.Context, proto.Message) ([]proto.Message, error) {
	return nil, orm.ErrNotImplemented
}

func (r *Redis) Delete(ctx context.Context, value proto.Message) error {
	if r.client == nil {
		return orm.ErrDbDriverNotInit
	}
	tm := meta.GetMetaByValue(value)
	if tm == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, value); err != nil {
		return err
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	key, err := recordKey(tableName(dbObj), pk)
	if err != nil {
		return err
	}
	n, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return orm.ErrRecordNotFound
	}
	return nil
}

func tableName(dbObj proto.Message) string {
	if t, ok := dbObj.(interface{ TableName() string }); ok {
		return t.TableName()
	}
	return string(dbObj.ProtoReflect().Descriptor().Name())
}
