package sql

import (
	"context"
	"errors"
	"testing"

	"github.com/wxdqing/go-orm/orm"
)

type fullScanIndexProvider struct {
	indexes map[string]any
}

func (p fullScanIndexProvider) Indexes() []any             { return nil }
func (p fullScanIndexProvider) IndexNames() []string       { return nil }
func (p fullScanIndexProvider) ToIndexMap() map[string]any { return p.indexes }
func (p fullScanIndexProvider) ToIndexStruct() any         { return nil }

func TestFindIndexRejectsImplicitFullScan(t *testing.T) {
	_, err := findIndex(context.Background(), fullScanIndexProvider{})
	if !errors.Is(err, orm.ErrNoIndexSpecified) {
		t.Fatalf("findIndex error = %v, want %v", err, orm.ErrNoIndexSpecified)
	}
}

func TestFindIndexAllowsExplicitFullScan(t *testing.T) {
	indexes, err := findIndex(orm.WithFullScan(context.Background()), fullScanIndexProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 0 {
		t.Fatalf("indexes = %v, want empty", indexes)
	}
}

func TestFindIndexAllowsIndexedQuery(t *testing.T) {
	want := map[string]any{"account": "test"}
	indexes, err := findIndex(context.Background(), fullScanIndexProvider{indexes: want})
	if err != nil {
		t.Fatal(err)
	}
	if indexes["account"] != want["account"] {
		t.Fatalf("indexes = %v, want %v", indexes, want)
	}
}
