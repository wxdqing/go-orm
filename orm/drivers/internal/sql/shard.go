package sql

import (
	"fmt"

	"github.com/wxdqing/go-orm/orm"
	logger "git.wxdqing.com/sprout/logger.git"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type gormShardRouter struct {
	mode     orm.ShardMode
	keyField string
	policy   string
	tables   []orm.TableShardRule
	shardDBs []*gorm.DB
	primary  *gorm.DB
}

func newGormShardRouter(primary *gorm.DB, shardDBs []*gorm.DB, conf orm.SQLShardConf) gormShardRouter {
	conf.Normalize()
	return gormShardRouter{
		mode:     conf.Mode,
		keyField: conf.KeyField,
		policy:   conf.Policy,
		tables:   conf.Tables,
		shardDBs: shardDBs,
		primary:  primary,
	}
}

func (r gormShardRouter) tableSharding(baseTable string) bool {
	rule := r.tableRule(baseTable)
	return rule != nil && rule.ShardCount > 1
}

func (r gormShardRouter) combinedMode(baseTable string) bool {
	return r.databaseMode() && r.tableSharding(baseTable)
}

func (r gormShardRouter) databaseMode() bool {
	return r.mode == orm.ShardModeDatabase && len(r.shardDBs) > 0
}

func (r gormShardRouter) tableMode(baseTable string) bool {
	if r.mode != orm.ShardModeTable {
		return false
	}
	rule := r.tableRule(baseTable)
	return rule != nil && rule.ShardCount > 1
}

func (r gormShardRouter) tableRule(baseTable string) *orm.TableShardRule {
	for i := range r.tables {
		if r.tables[i].Table == baseTable {
			return &r.tables[i]
		}
	}
	return nil
}

func (r gormShardRouter) keyFieldFor(baseTable string) string {
	if rule := r.tableRule(baseTable); rule != nil && rule.KeyField != "" {
		return rule.KeyField
	}
	return r.keyField
}

func (r gormShardRouter) shardIndex(value proto.Message, baseTable string) (int, error) {
	count := 1
	if r.mode == orm.ShardModeDatabase || r.combinedMode(baseTable) {
		count = len(r.shardDBs)
	} else if rule := r.tableRule(baseTable); rule != nil {
		count = rule.ShardCount
	}
	field := r.keyFieldFor(baseTable)
	key, err := orm.ExtractShardKey(value, field)
	if err != nil {
		if r.mode == orm.ShardModeDatabase && (r.policy == orm.ShardPolicyRandom || r.policy == orm.ShardPolicyRoundRobin) {
			return orm.SelectShardIndex(0, count, r.policy)
		}
		return 0, err
	}
	if r.mode == orm.ShardModeDatabase || r.combinedMode(baseTable) {
		return orm.SelectShardIndex(key, count, r.policy)
	}
	return orm.ResolveShardIndex(key, count), nil
}

func (r gormShardRouter) db(value proto.Message, baseTable string) (*gorm.DB, error) {
	if r.databaseMode() || r.combinedMode(baseTable) {
		idx, err := r.shardIndex(value, baseTable)
		if err != nil {
			return nil, err
		}
		if idx >= 0 && idx < len(r.shardDBs) {
			return r.shardDBs[idx], nil
		}
	}
	return r.primary, nil
}

func (r gormShardRouter) session(value proto.Message, baseTable string) (*gorm.DB, error) {
	db, err := r.db(value, baseTable)
	if err != nil {
		return nil, err
	}
	if r.mode != orm.ShardModeTable && !r.combinedMode(baseTable) {
		return db, nil
	}
	rule := r.tableRule(baseTable)
	if rule == nil || rule.ShardCount <= 1 {
		return db, nil
	}
	field := r.keyFieldFor(baseTable)
	key, err := orm.ExtractShardKey(value, field)
	if err != nil {
		return nil, err
	}
	table := orm.ResolveTableName(baseTable, key, *rule)
	logger.Debugf("gorm shard table route: %s -> %s", baseTable, table)
	return db.Table(table), nil
}

func shardPhysicalTables(baseTable string, rule *orm.TableShardRule) []string {
	if rule == nil || rule.ShardCount <= 1 {
		return []string{baseTable}
	}
	names := make([]string, 0, rule.ShardCount)
	for i := 0; i < rule.ShardCount; i++ {
		suffix := fmt.Sprintf(rule.SuffixFormat, i)
		names = append(names, baseTable+suffix)
	}
	return names
}
