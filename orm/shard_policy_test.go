package orm

import "testing"

func TestSelectShardIndex_Hash(t *testing.T) {
	idx, err := SelectShardIndex(7, 4, ShardPolicyHash)
	if err != nil || idx != 3 {
		t.Fatalf("hash idx=%d err=%v", idx, err)
	}
}

func TestSelectShardIndex_RoundRobin(t *testing.T) {
	shardRoundRobinCounter = 0
	seen := make(map[int]bool)
	for i := 0; i < 8; i++ {
		idx, err := SelectShardIndex(0, 4, ShardPolicyRoundRobin)
		if err != nil {
			t.Fatal(err)
		}
		seen[idx] = true
	}
	if len(seen) < 2 {
		t.Fatal("round_robin should rotate indices")
	}
}

func TestSQLShardConf_Validate_Policy(t *testing.T) {
	c := SQLShardConf{Mode: ShardModeNone, Policy: "invalid"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected policy error")
	}
}
