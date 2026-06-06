//go:build db

package logic

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
	"gs/pbtest/metadata"
	"google.golang.org/protobuf/proto"
)

func coverageTestID(t *testing.T) int64 {
	t.Helper()
	return 950_000_000 + (time.Now().UnixNano() % 1_000_000)
}

func closeDriverAfterTest(t *testing.T) {
	t.Helper()
	_ = drivers.Close(context.Background())
	t.Cleanup(func() { _ = drivers.Close(context.Background()) })
}

func tableByName(driverType, name string) proto.Message {
	for _, tb := range metadata.GetAllTables(driverType) {
		if reflect.TypeOf(tb).Elem().Name() == name {
			return tb
		}
	}
	return nil
}

func tablesByNames(driverType string, names ...string) []proto.Message {
	out := make([]proto.Message, 0, len(names))
	for _, name := range names {
		if tb := tableByName(driverType, name); tb != nil {
			out = append(out, tb)
		}
	}
	return out
}

func tryInitDriver(t *testing.T, driverType string, conf *orm.Conf, tableNames ...string) {
	t.Helper()
	closeDriverAfterTest(t)
	tables := metadata.GetAllTables(driverType)
	if len(tableNames) > 0 {
		tables = tablesByNames(driverType, tableNames...)
		if len(tables) != len(tableNames) {
			t.Fatalf("driver %s: want tables %v", driverType, tableNames)
		}
	}
	if err := drivers.TryInit(context.Background(),
		drivers.WithDriverType(driverType),
		drivers.WithTables(tables),
		drivers.WithConfig(conf),
	); err != nil {
		t.Fatal(err)
	}
}

func sqlCoverageTableNames() []string {
	return []string{"FieldsPlayer", "GameRole", "Lister", "Player"}
}

func kvCoverageTableNames() []string {
	return []string{"Player", "VersionedPlayer", "FieldsPlayer"}
}

func sampleFieldsPlayer(id int64) *pbtest.FieldsPlayer {
	return &pbtest.FieldsPlayer{
		Id:         id,
		Name:       "fields_coverage",
		Level:      11,
		Exp:        99,
		PlayerEnum: pbtest.CoveragePlayerEnum_CoveragePlayerEnum_Test1,
		Settings:   map[string]int32{"k": 42},
		Heros:      []*pbtest.FieldsHero{{Id: 1, Cid: 3, HeroLevel: 5}},
	}
}

func assertFieldsPlayer(t *testing.T, got, want *pbtest.FieldsPlayer) {
	t.Helper()
	if got.Name != want.Name || got.Level != want.Level || got.Exp != want.Exp {
		t.Fatalf("scalar: got %+v want %+v", got, want)
	}
	if got.PlayerEnum != want.PlayerEnum {
		t.Fatalf("enum: got %v want %v", got.PlayerEnum, want.PlayerEnum)
	}
	if len(got.Settings) != 1 || got.Settings["k"] != 42 {
		t.Fatalf("settings: %+v", got.Settings)
	}
	if len(got.Heros) != 1 || got.Heros[0].Cid != 3 {
		t.Fatalf("heros: %+v", got.Heros)
	}
}

func sampleGameRole(serverID, roleID int64) *pbtest.GameRole {
	return &pbtest.GameRole{
		ServerId:   serverID,
		RoleId:     roleID,
		Name:       "role_alpha",
		Level:      15,
		Exp:        500,
		Heros:      []*pbtest.RoleHero{{Id: 2, Cid: 7, HeroLevel: 4}},
		Timestamps: &pbtest.RoleTimestamps{CreatedAt: 1_700_000_000_000, UpdatedAt: 1_700_000_100_000},
		Settings:   map[string]int32{"zone": 8},
		PlayerEnum: pbtest.CoveragePlayerEnum_CoveragePlayerEnum_Test1,
		Profile:    []byte(`{"title":"knight"}`),
	}
}

func assertGameRole(t *testing.T, got, want *pbtest.GameRole) {
	t.Helper()
	if got.ServerId != want.ServerId || got.RoleId != want.RoleId {
		t.Fatalf("pk: got (%d,%d) want (%d,%d)", got.ServerId, got.RoleId, want.ServerId, want.RoleId)
	}
	if got.Name != want.Name || got.Level != want.Level {
		t.Fatalf("fields: got %+v want %+v", got, want)
	}
	if len(got.Heros) != 1 || got.Settings["zone"] != 8 {
		t.Fatalf("blob/json: heros=%+v settings=%+v", got.Heros, got.Settings)
	}
	if len(got.Profile) == 0 {
		t.Fatalf("profile empty")
	}
	if got.Timestamps == nil || got.Timestamps.CreatedAt != want.Timestamps.CreatedAt {
		t.Fatalf("embed timestamps: got %+v want %+v", got.Timestamps, want.Timestamps)
	}
}

func sampleCoveragePlayer(id int64) *pbtest.Player {
	return &pbtest.Player{
		Id:    id,
		Name:  "payload_coverage",
		Level: 42,
		BaseInfo: &pbtest.RoleBaseInfo{
			Rid:      uint64(id),
			NickName: "nick",
			Level:    42,
		},
		ItemInfo: &pbtest.ItemInfo{
			Items: map[int32]*pbtest.Item{1: {ConfigId: 100, Num: 3}},
		},
	}
}

func assertCoveragePlayer(t *testing.T, got, want *pbtest.Player) {
	t.Helper()
	if got.Name != want.Name || got.Level != want.Level {
		t.Fatalf("scalar: got %+v want %+v", got, want)
	}
	if got.BaseInfo == nil || got.BaseInfo.NickName != "nick" {
		t.Fatalf("base_info: %+v", got.BaseInfo)
	}
	if got.ItemInfo == nil || got.ItemInfo.Items[1].Num != 3 {
		t.Fatalf("item_info: %+v", got.ItemInfo)
	}
}

func testFieldsPlayerCRUD(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitDriver(t, driverType, conf, "FieldsPlayer")
	id := coverageTestID(t)
	want := sampleFieldsPlayer(id)
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.FieldsPlayer{Id: id})
	})
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &pbtest.FieldsPlayer{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertFieldsPlayer(t, got, want)
	if err := drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.FieldsPlayer{Id: id}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatalf("after Delete: %v", err)
	}
}

func testGameRoleCompositeCRUD(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitDriver(t, driverType, conf, sqlCoverageTableNames()...)
	serverID := int64(1001)
	roleID := coverageTestID(t)
	want := sampleGameRole(serverID, roleID)
	key := &pbtest.GameRole{ServerId: serverID, RoleId: roleID}
	t.Cleanup(func() { _ = drivers.DefaultDbDriver.Delete(context.Background(), key) })
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &pbtest.GameRole{ServerId: serverID, RoleId: roleID}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertGameRole(t, got, want)
	if err := drivers.DefaultDbDriver.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func testGameRoleFindByName(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitDriver(t, driverType, conf, sqlCoverageTableNames()...)
	serverID := int64(3003)
	roleID := coverageTestID(t)
	want := sampleGameRole(serverID, roleID)
	want.Name = "find_me_role"
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.GameRole{ServerId: serverID, RoleId: roleID})
	})
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	rows, err := drivers.DefaultDbDriver.Find(context.Background(), &pbtest.GameRole{Name: "find_me_role"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	found := false
	for _, r := range rows {
		p, ok := r.(*pbtest.GameRole)
		if !ok {
			continue
		}
		if p.ServerId == serverID && p.RoleId == roleID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Find: missing saved role among %d rows", len(rows))
	}
}

func testListerCompositeCRUD(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitDriver(t, driverType, conf, sqlCoverageTableNames()...)
	rid := int64(2002)
	id := coverageTestID(t)
	want := &pbtest.Lister{Rid: rid, Id: id, Data: []byte("offline-payload")}
	key := &pbtest.Lister{Rid: rid, Id: id}
	t.Cleanup(func() { _ = drivers.DefaultDbDriver.Delete(context.Background(), key) })
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &pbtest.Lister{Rid: rid, Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.Data) != string(want.Data) {
		t.Fatalf("data: %q want %q", got.Data, want.Data)
	}
}

func testPlayerPayloadCRUD(t *testing.T, driverType string, conf *orm.Conf, tableNames ...string) {
	t.Helper()
	if len(tableNames) == 0 {
		tableNames = []string{"Player"}
	}
	tryInitDriver(t, driverType, conf, tableNames...)
	id := coverageTestID(t)
	want := sampleCoveragePlayer(id)
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.Player{Id: id})
	})
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &pbtest.Player{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertCoveragePlayer(t, got, want)
	if err := drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.Player{Id: id}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
