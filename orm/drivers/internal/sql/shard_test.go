package sql

import (
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"gorm.io/gorm"
)

func TestShardPhysicalTables(t *testing.T) {
	rule := &orm.TableShardRule{ShardCount: 4, SuffixFormat: "_%d"}
	got := shardPhysicalTables("player", rule)
	want := []string{"player_0", "player_1", "player_2", "player_3"}
	if len(got) != len(want) {
		t.Fatalf("tables = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tables[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGormShardRouterTableMode(t *testing.T) {
	r := newGormShardRouter(nil, nil, orm.SQLShardConf{
		Mode:     orm.ShardModeTable,
		KeyField: "id",
		Tables: []orm.TableShardRule{
			{Table: "player", ShardCount: 4},
		},
	})
	if !r.tableMode("player") {
		t.Fatal("player should be in table shard mode")
	}
	if r.tableMode("other") {
		t.Fatal("unknown table should not be sharded")
	}
}

func TestGormShardRouter_CombinedDatabaseAndTable(t *testing.T) {
	r := newGormShardRouter(nil, []*gorm.DB{{}, {}, {}, {}}, orm.SQLShardConf{
		Mode:     orm.ShardModeDatabase,
		KeyField: "id",
		Policy:   orm.ShardPolicyHash,
		Tables: []orm.TableShardRule{
			{Table: "player", ShardCount: 4, SuffixFormat: "_%d"},
		},
	})
	if !r.combinedMode("player") {
		t.Fatal("expected combined mode for player")
	}
}

func TestGormShardRouterKeyFieldFor(t *testing.T) {
	r := newGormShardRouter(nil, nil, orm.SQLShardConf{
		Mode:     orm.ShardModeTable,
		KeyField: "id",
		Tables: []orm.TableShardRule{
			{Table: "player", KeyField: "user_id", ShardCount: 2},
		},
	})
	if got := r.keyFieldFor("player"); got != "user_id" {
		t.Fatalf("keyFieldFor(player) = %q, want user_id", got)
	}
	if got := r.keyFieldFor("guild"); got != "id" {
		t.Fatalf("keyFieldFor(guild) = %q, want id", got)
	}
}
