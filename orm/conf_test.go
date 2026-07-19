package orm

import (
	"strings"
	"testing"
)

func TestDecodeMapToStructWithShardAndStartup(t *testing.T) {
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
							"table":         "player",
							"shard_count":   4,
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

func TestDecodeMapToStructNilUsesDefaults(t *testing.T) {
	c := &Conf{}
	if err := DecodeMapToStruct(nil, c); err != nil {
		t.Fatal(err)
	}
	if c.Driver != "mysql" || c.Mysql.Addr == "" || c.Redis.Host == "" {
		t.Fatalf("nil config did not load defaults: %+v", c)
	}
}

func TestDecodeMapToStructCallsAreIndependent(t *testing.T) {
	first := &Conf{}
	if err := DecodeMapToStruct(map[string]any{"db": map[string]any{
		"redis": map[string]any{"host": "redis.internal:6379", "password": "secret"},
	}}, first); err != nil {
		t.Fatal(err)
	}
	second := &Conf{}
	if err := DecodeMapToStruct(map[string]any{"db": map[string]any{
		"driver": "mysql",
	}}, second); err != nil {
		t.Fatal(err)
	}
	if second.Redis.Host == first.Redis.Host || second.Redis.Password != "" {
		t.Fatalf("second decode inherited first redis config: %+v", second.Redis)
	}
}

func TestDecodeMapToStructReturnsDecodeError(t *testing.T) {
	c := &Conf{}
	err := DecodeMapToStruct(map[string]any{"db": map[string]any{
		"mysql": map[string]any{"max_open": []string{"invalid"}},
	}}, c)
	if err == nil || !strings.Contains(err.Error(), "mysql") {
		t.Fatalf("DecodeMapToStruct() error = %v, want mysql decode error", err)
	}
}

func TestDecodeMapToStructRejectsNilTarget(t *testing.T) {
	if err := DecodeMapToStruct(map[string]any{}, nil); err == nil {
		t.Fatal("DecodeMapToStruct() error = nil, want invalid target error")
	}
}
