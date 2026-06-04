package orm

import (
	"fmt"
	"hash/fnv"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ShardingKeyProvider 业务值对象可提供分片键（可选，优先于配置字段名）。
type ShardingKeyProvider interface {
	ShardingKey() int64
}

// ResolveShardIndex 根据分片键计算下标 [0, shardCount)。
func ResolveShardIndex(shardKey int64, shardCount int) int {
	if shardCount <= 1 {
		return 0
	}
	if shardKey < 0 {
		shardKey = -shardKey
	}
	return int(shardKey % int64(shardCount))
}

// ResolveTableName 解析分表物理表名。
func ResolveTableName(baseTable string, shardKey int64, rule TableShardRule) string {
	if rule.ShardCount <= 1 {
		return baseTable
	}
	idx := ResolveShardIndex(shardKey, rule.ShardCount)
	suffix := fmt.Sprintf(rule.SuffixFormat, idx)
	return baseTable + suffix
}

// ExtractShardKey 从 proto 消息提取分片键。
func ExtractShardKey(msg proto.Message, fieldName string) (int64, error) {
	if msg == nil {
		return 0, fmt.Errorf("shard: message is nil")
	}
	if p, ok := msg.(ShardingKeyProvider); ok {
		return p.ShardingKey(), nil
	}
	if fieldName == "" {
		return 0, fmt.Errorf("shard: key field not configured")
	}
	rv := msg.ProtoReflect()
	fd := rv.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		return 0, fmt.Errorf("shard: field %q not found on %s", fieldName, rv.Descriptor().FullName())
	}
	switch fd.Kind() {
	case protoreflect.Int32Kind:
		return int64(rv.Get(fd).Int()), nil
	case protoreflect.Int64Kind:
		return rv.Get(fd).Int(), nil
	case protoreflect.Uint32Kind:
		return int64(rv.Get(fd).Uint()), nil
	case protoreflect.Uint64Kind:
		return int64(rv.Get(fd).Uint()), nil
	default:
		return 0, fmt.Errorf("shard: field %q is not integer", fieldName)
	}
}

// HashShardKey 将任意类型分片键归一为 int64（字符串等）。
func HashShardKey(key any) int64 {
	switch v := key.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case string:
		h := fnv.New64a()
		_, _ = h.Write([]byte(v))
		return int64(h.Sum64())
	default:
		rv := reflect.ValueOf(key)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int64(rv.Uint())
		default:
			return 0
		}
	}
}
