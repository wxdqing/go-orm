package kv

import (
	"testing"

	"github.com/wxdqing/go-orm/orm"
)

func TestRecordKey_SortedStable(t *testing.T) {
	k, err := recordKey("player", map[string]any{"id": int64(9), "zone": int32(1)})
	if err != nil {
		t.Fatal(err)
	}
	want := "player:id=9:zone=1"
	if k != want {
		t.Fatalf("recordKey() = %q, want %q", k, want)
	}
}

func TestRecordKey_NoPrimaryKey(t *testing.T) {
	_, err := recordKey("player", nil)
	if err != orm.ErrNoPrimaryKeySpecified {
		t.Fatalf("err = %v, want ErrNoPrimaryKeySpecified", err)
	}
}
