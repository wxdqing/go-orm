package orm

import "fmt"

// Validate 校验分片配置（Normalize 之后调用）。
func (c *SQLShardConf) Validate() error {
	c.Normalize()
	if err := ValidateShardPolicy(c.Policy); err != nil {
		return err
	}
	switch c.Mode {
	case ShardModeNone:
		return nil
	case ShardModeTable:
		if c.KeyField == "" {
			return fmt.Errorf("%w: table shard requires key_field", ErrInvalidDriverOptions)
		}
		if len(c.Tables) == 0 {
			return fmt.Errorf("%w: table shard requires tables", ErrInvalidDriverOptions)
		}
		for i, rule := range c.Tables {
			if rule.Table == "" {
				return fmt.Errorf("%w: tables[%d].table is empty", ErrInvalidDriverOptions, i)
			}
			if rule.ShardCount < 2 {
				return fmt.Errorf("%w: tables[%d].shard_count must be >= 2", ErrInvalidDriverOptions, i)
			}
		}
		return nil
	case ShardModeDatabase:
		if c.KeyField == "" {
			return fmt.Errorf("%w: database shard requires key_field", ErrInvalidDriverOptions)
		}
		if len(c.Sources) == 0 {
			return fmt.Errorf("%w: database shard requires sources", ErrInvalidDriverOptions)
		}
		if c.Policy != ShardPolicyHash {
			return fmt.Errorf("%w: database shard requires hash policy, got %q", ErrInvalidDriverOptions, c.Policy)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown shard mode %q", ErrInvalidDriverOptions, c.Mode)
	}
}
