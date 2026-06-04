package orm

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"
)

// ShardPolicy 分库源选择策略（database 模式）。
const (
	ShardPolicyHash        = "hash"         // 默认：按分片键取模
	ShardPolicyRandom      = "random"       // 随机分库（无分片键时或显式 random）
	ShardPolicyRoundRobin  = "round_robin"  // 轮询分库
)

var shardRoundRobinCounter uint64

// SelectShardIndex 根据策略计算分库下标 [0, shardCount)。
func SelectShardIndex(shardKey int64, shardCount int, policy string) (int, error) {
	if shardCount <= 1 {
		return 0, nil
	}
	switch policy {
	case "", ShardPolicyHash:
		return ResolveShardIndex(shardKey, shardCount), nil
	case ShardPolicyRandom:
		return rand.IntN(shardCount), nil
	case ShardPolicyRoundRobin:
		n := atomic.AddUint64(&shardRoundRobinCounter, 1)
		return int((n - 1) % uint64(shardCount)), nil
	default:
		return 0, fmt.Errorf("%w: unknown shard policy %q", ErrInvalidDriverOptions, policy)
	}
}

// ValidateShardPolicy 校验 policy 字段。
func ValidateShardPolicy(policy string) error {
	switch policy {
	case "", ShardPolicyHash, ShardPolicyRandom, ShardPolicyRoundRobin:
		return nil
	default:
		return fmt.Errorf("%w: unknown shard policy %q", ErrInvalidDriverOptions, policy)
	}
}
