package orm

import "testing"

func TestResolveTableName(t *testing.T) {
	rule := TableShardRule{ShardCount: 4, SuffixFormat: "_%d"}
	if got := ResolveTableName("player", 7, rule); got != "player_3" {
		t.Fatalf("ResolveTableName() = %q, want player_3", got)
	}
}

func TestResolveShardIndex(t *testing.T) {
	if ResolveShardIndex(10, 4) != 2 {
		t.Fatalf("index = %d, want 2", ResolveShardIndex(10, 4))
	}
	if ResolveShardIndex(1, 4) != 1 {
		t.Fatalf("index(1,4) = %d, want 1", ResolveShardIndex(1, 4))
	}
	if ResolveShardIndex(4, 4) != 0 {
		t.Fatalf("index(4,4) = %d, want 0", ResolveShardIndex(4, 4))
	}
}
