package orm

import "testing"

func TestSQLShardConf_Validate_TableModeRequiresCount(t *testing.T) {
	c := SQLShardConf{
		Mode:     ShardModeTable,
		KeyField: "id",
		Tables:   []TableShardRule{{Table: "player", ShardCount: 1}},
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for shard_count < 2")
	}
}

func TestSQLShardConf_Validate_DatabaseRequiresSources(t *testing.T) {
	c := SQLShardConf{
		Mode:     ShardModeDatabase,
		KeyField: "id",
		Sources:  nil,
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty sources")
	}
}

func TestSQLShardConf_Validate_NoneOK(t *testing.T) {
	c := SQLShardConf{Mode: ShardModeNone}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLShardConf_Validate_KeyFieldRequired(t *testing.T) {
	table := SQLShardConf{
		Mode:   ShardModeTable,
		Tables: []TableShardRule{{Table: "player", ShardCount: 2}},
	}
	if err := table.Validate(); err == nil {
		t.Fatal("expected error when table mode missing key_field")
	}
	db := SQLShardConf{
		Mode:    ShardModeDatabase,
		Sources: []DatabaseShardSource{{Name: "shard0"}},
	}
	if err := db.Validate(); err == nil {
		t.Fatal("expected error when database mode missing key_field")
	}
}
