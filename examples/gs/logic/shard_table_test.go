package logic

import (
	"context"
	"fmt"
	"testing"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
	logger "git.wxdqing.com/sprout/logger.git"
)

func playerShardIndex(id int64) int {
	return orm.ResolveShardIndex(id, shardTestCount)
}

func physicalPlayerTable(id int64) string {
	return fmt.Sprintf("player_%d", playerShardIndex(id))
}

func countPlayerInTable(t *testing.T, table string, id int64) int64 {
	t.Helper()
	db := drivers.ToGorm()
	if db == nil {
		t.Fatal("gorm db is nil")
	}
	var n int64
	if err := db.Table(table).Where("id = ?", id).Count(&n).Error; err != nil {
		t.Fatalf("count %s id=%d: %v", table, id, err)
	}
	return n
}

func cleanupShardTestPlayers(t *testing.T, ids ...int64) {
	t.Helper()
	for _, id := range ids {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.Player{Id: id})
	}
}

func TestMysqlTableShardSaveAndRoute(t *testing.T) {
	logger.Init()
	if err := UseMysqlDriverWithTableShard(); err != nil {
		t.Fatal(err)
	}

	const id int64 = 88001
	cleanupShardTestPlayers(t, id)
	t.Cleanup(func() { cleanupShardTestPlayers(t, id) })

	p := &pbtest.Player{Id: id, Name: "shard_mysql"}
	if err := drivers.DefaultDbDriver.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	wantTable := physicalPlayerTable(id)
	for i := 0; i < shardTestCount; i++ {
		table := fmt.Sprintf("player_%d", i)
		n := countPlayerInTable(t, table, id)
		if table == wantTable {
			if n != 1 {
				t.Fatalf("id %d should exist in %s, count=%d", id, table, n)
			}
		} else if n != 0 {
			t.Fatalf("id %d should not exist in %s, count=%d", id, table, n)
		}
	}

	got := &pbtest.Player{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "shard_mysql" {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestMysqlTableShardFind(t *testing.T) {
	logger.Init()
	if err := UseMysqlDriverWithTableShard(); err != nil {
		t.Fatal(err)
	}

	ids := []int64{88011, 88012, 88013, 88014, 88015}
	cleanupShardTestPlayers(t, ids...)
	t.Cleanup(func() { cleanupShardTestPlayers(t, ids...) })

	for _, id := range ids {
		name := fmt.Sprintf("shard_find_%d", id)
		if err := drivers.DefaultDbDriver.Save(context.Background(), &pbtest.Player{Id: id, Name: name}); err != nil {
			t.Fatal(err)
		}
		got := &pbtest.Player{Id: id}
		if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
			t.Fatal(err)
		}
		if got.Name != name {
			t.Fatalf("id %d name = %q, want %q", id, got.Name, name)
		}
	}

	// 按索引条件查询（仅命中单个分片上的记录）
	list, err := drivers.DefaultDbDriver.Find(context.Background(), &pbtest.Player{Name: "shard_find_88012"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("Find by name: got %d rows, want 1", len(list))
	}
}

func TestPgsqlTableShardSaveAndRoute(t *testing.T) {
	logger.Init()
	if err := UsePgsqlDriverWithTableShard(); err != nil {
		t.Fatal(err)
	}

	const id int64 = 89001
	cleanupShardTestPlayers(t, id)
	t.Cleanup(func() { cleanupShardTestPlayers(t, id) })

	p := &pbtest.Player{Id: id, Name: "shard_pgsql"}
	if err := drivers.DefaultDbDriver.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	wantTable := physicalPlayerTable(id)
	for i := 0; i < shardTestCount; i++ {
		table := fmt.Sprintf("player_%d", i)
		n := countPlayerInTable(t, table, id)
		if table == wantTable {
			if n != 1 {
				t.Fatalf("id %d should exist in %s, count=%d", id, table, n)
			}
		} else if n != 0 {
			t.Fatalf("id %d should not exist in %s, count=%d", id, table, n)
		}
	}

	got := &pbtest.Player{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "shard_pgsql" {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestPgsqlTableShardFind(t *testing.T) {
	logger.Init()
	if err := UsePgsqlDriverWithTableShard(); err != nil {
		t.Fatal(err)
	}

	ids := []int64{89011, 89012, 89013, 89014}
	cleanupShardTestPlayers(t, ids...)
	t.Cleanup(func() { cleanupShardTestPlayers(t, ids...) })

	for _, id := range ids {
		name := fmt.Sprintf("pg_find_%d", id)
		if err := drivers.DefaultDbDriver.Save(context.Background(), &pbtest.Player{Id: id, Name: name}); err != nil {
			t.Fatal(err)
		}
		got := &pbtest.Player{Id: id}
		if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
			t.Fatal(err)
		}
		if got.Name != name {
			t.Fatalf("id %d name = %q, want %q", id, got.Name, name)
		}
	}

	list, err := drivers.DefaultDbDriver.Find(context.Background(), &pbtest.Player{Name: "pg_find_89012"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("Find by name: got %d rows, want 1", len(list))
	}
}

