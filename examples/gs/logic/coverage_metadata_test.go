package logic

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
	"gs/pbtest/metadata"
)

// EX-GEN-003 — 新表已在四类 backend metadata 注册。
func TestCoverage_Metadata_AllBackends(t *testing.T) {
	expect := map[string][]string{
		drivers.DriverTypeMySQL:       {"FieldsPlayer", "GameRole", "Lister", "Player", "VersionedPlayer"},
		drivers.DriverTypePostgresSQL: {"FieldsPlayer", "GameRole", "Lister", "Player", "VersionedPlayer"},
		drivers.DriverTypeRedis:       {"FieldsPlayer", "GameRole", "Lister", "Player", "VersionedPlayer"},
		drivers.DriverTypeMongoDB:     {"FieldsPlayer", "GameRole", "Lister", "Player", "VersionedPlayer"},
	}
	for dt, names := range expect {
		tables := metadata.GetAllTables(dt)
		if len(tables) < len(names) {
			t.Fatalf("%s: got %d tables, want at least %d", dt, len(tables), len(names))
		}
		got := make(map[string]bool)
		for _, tb := range tables {
			got[reflect.TypeOf(tb).Elem().Name()] = true
		}
		for _, name := range names {
			if !got[name] {
				t.Fatalf("%s: missing table %s in metadata", dt, name)
			}
		}
	}
}

// EX-RD-004 — FIELDS 表在 KV 生成物仍为 PAYLOAD 形态（含 data 字段、无列展开）。
func TestCoverage_KV_FieldsPlayer_IsPayloadShape(t *testing.T) {
	for _, dt := range []string{drivers.DriverTypeRedis, drivers.DriverTypeMongoDB} {
		var fp reflect.Type
		for _, tb := range metadata.GetAllTables(dt) {
			if reflect.TypeOf(tb).Elem().Name() == "FieldsPlayer" {
				fp = reflect.TypeOf(tb).Elem()
				break
			}
		}
		if fp == nil {
			t.Fatalf("%s: FieldsPlayer not in metadata", dt)
		}
		if _, ok := fp.FieldByName("Data"); !ok {
			t.Fatalf("%s FieldsPlayer: missing data field for KV PAYLOAD", dt)
		}
		if _, ok := fp.FieldByName("Settings"); ok {
			t.Fatalf("%s FieldsPlayer: should not expose FIELDS column Settings", dt)
		}
	}
}

// EX-GEN-004 — MySQL/PG GameRole 含 embedded gorm tag。
func TestCoverage_SQL_GameRole_HasEmbeddedTag(t *testing.T) {
	for _, dt := range []string{drivers.DriverTypeMySQL, drivers.DriverTypePostgresSQL} {
		var gr reflect.Type
		for _, tb := range metadata.GetAllTables(dt) {
			if reflect.TypeOf(tb).Elem().Name() == "GameRole" {
				gr = reflect.TypeOf(tb).Elem()
				break
			}
		}
		if gr == nil {
			t.Fatalf("%s: GameRole not in metadata", dt)
		}
		f, ok := gr.FieldByName("Timestamps")
		if !ok {
			t.Fatalf("%s GameRole: missing Timestamps field", dt)
		}
		tag := f.Tag.Get("gorm")
		if !strings.Contains(tag, "embedded") {
			t.Fatalf("%s GameRole Timestamps gorm tag = %q, want embedded", dt, tag)
		}
	}
}

// EX-SC-006 — tables_tags.proto 已生成 TagExample（含 tags / oneof 字段）。
func TestCoverage_TagExample_SchemaPresent(t *testing.T) {
	msg := pbtest.File_tables_tags_proto.Messages().ByName("TagExample")
	if msg == nil {
		t.Fatal("TagExample not in generated descriptor")
	}
	if msg.Fields().ByName("with_tags") == nil {
		t.Fatal("missing with_tags field")
	}
	if msg.Fields().ByName("a") == nil || msg.Fields().ByName("a").ContainingOneof() == nil {
		t.Fatal("missing oneof field a")
	}
}
