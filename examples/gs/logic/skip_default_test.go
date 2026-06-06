package logic

import (
	"testing"

	"gs/pbtest"
)

// EX-MY-012 / EX-PG-012 — skip_set_default 不写入 SetDefaults。
func TestFieldsPlayer_SkipSetDefault_NotInSetDefaults(t *testing.T) {
	r := &pbtest.FieldsPlayer{}
	pbtest.SetFieldsPlayerDefaults(r)
	if r.SkipMe != nil {
		t.Fatalf("SkipMe should remain nil after SetDefaults, got %+v", r.SkipMe)
	}
}
