//go:build db

package logic

import (
	"context"
	"testing"

	"github.com/wxdqing/go-orm/orm/drivers"
	"gs/pbtest"
)

func TestIntegration_SkipDefault_MySQL_FieldsPlayer(t *testing.T) {
	tryInitDriver(t, drivers.DriverTypeMySQL, mysqlConf(), "FieldsPlayer")
	id := coverageTestID(t)
	want := sampleFieldsPlayer(id)
	want.SkipMe = nil
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.FieldsPlayer{Id: id})
	})
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got := &pbtest.FieldsPlayer{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if got.SkipMe != nil {
		t.Fatalf("SkipMe persisted unexpectedly: %+v", got.SkipMe)
	}
}

func TestIntegration_SkipDefault_Pgsql_FieldsPlayer(t *testing.T) {
	tryInitDriver(t, drivers.DriverTypePostgresSQL, pgsqlConf(), "FieldsPlayer")
	id := coverageTestID(t)
	want := sampleFieldsPlayer(id)
	want.SkipMe = nil
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &pbtest.FieldsPlayer{Id: id})
	})
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got := &pbtest.FieldsPlayer{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if got.SkipMe != nil {
		t.Fatalf("SkipMe persisted unexpectedly: %+v", got.SkipMe)
	}
}
