package sql

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type singlePKSaveRow struct {
	ID   int64 `gorm:"column:id;primaryKey"`
	Name string
}

func (singlePKSaveRow) TableName() string { return "single_save_tbl" }

func (r *singlePKSaveRow) PrimaryKey() []any { return []any{r.ID} }

func (r *singlePKSaveRow) PrimaryKeyNames() []string { return []string{"id"} }

func (r *singlePKSaveRow) ToPrimaryKeyMap() map[string]any { return map[string]any{"id": r.ID} }

func (r *singlePKSaveRow) ToPrimaryKeyStruct() any { return r }

// TestGormSave_SinglePKUsesSave 非 postgres 单主键可插入并更新（T-CR-004）。
func TestGormSave_SinglePKUsesSave(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&singlePKSaveRow{}); err != nil {
		t.Fatal(err)
	}
	row := &singlePKSaveRow{ID: 1, Name: "a"}
	if err := gormSaveRecord(db, row); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var got singlePKSaveRow
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "a" {
		t.Fatalf("got %+v", got)
	}
	row.Name = "b"
	if err := gormSaveRecord(db, row); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := db.First(&got, 1).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "b" {
		t.Fatalf("update got %+v", got)
	}
}

// TestGormSave_NoPkProviderFallsBackToSave 无 PkProvider 时直接 Save。
func TestGormSave_NoPkProviderFallsBackToSave(t *testing.T) {
	type plainRow struct {
		ID   int64 `gorm:"primaryKey"`
		Name string
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&plainRow{}); err != nil {
		t.Fatal(err)
	}
	row := &plainRow{ID: 9, Name: "x"}
	if err := gormSaveRecord(db, row); err != nil {
		t.Fatal(err)
	}
	var got plainRow
	if err := db.First(&got, 9).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "x" {
		t.Fatalf("got %+v", got)
	}
}
