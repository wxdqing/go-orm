package orm

import (
	"testing"

	logger "gitee.com/wxdqing/logger.git"
)

func TestDecodeMapToStructWithShardAndStartup(t *testing.T) {
	logger.Init()
	m := map[string]any{
		"db": map[string]any{
			"driver": "mysql",
			"mysql": map[string]any{
				"addr":     "localhost:3306",
				"name":     "game",
				"user":     "root",
				"password": "root123",
				"startup": map[string]any{
					"skip_default_transaction": true,
					"prepare_stmt":             true,
					"table_options":            "ENGINE=InnoDB",
					"time_zone":                "Asia/Shanghai",
				},
				"shard": map[string]any{
					"mode":      "table",
					"key_field": "id",
					"tables": []map[string]any{
						{
							"table":        "player",
							"shard_count":  4,
							"suffix_format": "_%d",
						},
					},
				},
			},
			"pgsql": map[string]any{
				"host":     "localhost",
				"port":     "5432",
				"name":     "game",
				"user":     "postgres",
				"password": "postgres123",
				"shard": map[string]any{
					"mode": "database",
					"sources": []map[string]any{
						{"name": "game_0", "host": "localhost", "port": "5432", "dbname": "game0", "user": "postgres", "password": "postgres123"},
						{"name": "game_1", "host": "localhost", "port": "5432", "dbname": "game1", "user": "postgres", "password": "postgres123"},
					},
				},
			},
		},
	}
	c := &Conf{}
	if err := DecodeMapToStruct(m, c); err != nil {
		t.Fatal(err)
	}
	if c.Mysql.Shard.Mode != ShardModeTable {
		t.Fatalf("mysql shard mode = %s", c.Mysql.Shard.Mode)
	}
	if c.Mysql.Startup.TableOptions != "ENGINE=InnoDB" {
		t.Fatalf("mysql startup table_options = %q", c.Mysql.Startup.TableOptions)
	}
	if c.Pgsql.Shard.Mode != ShardModeDatabase {
		t.Fatalf("pgsql shard mode = %s", c.Pgsql.Shard.Mode)
	}
	if len(c.Pgsql.Shard.Sources) != 2 {
		t.Fatalf("pgsql sources = %d", len(c.Pgsql.Shard.Sources))
	}
}
