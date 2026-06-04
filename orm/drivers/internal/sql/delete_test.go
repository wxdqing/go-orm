package sql

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type compositePKRow struct {
	ID       int64 `gorm:"column:id;primaryKey"`
	TenantID int64 `gorm:"column:tenant_id;primaryKey"`
}

func (compositePKRow) TableName() string { return "composite_tbl" }

func (r *compositePKRow) ToPrimaryKeyMap() map[string]any {
	return map[string]any{"id": r.ID, "tenant_id": r.TenantID}
}

// TestGormDelete_UsesPrimaryKeyMap 断言 Delete 路径使用 ToPrimaryKeyMap 而非写死 id（T-CR-001）。
func TestGormDelete_UsesPrimaryKeyMap(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	row := &compositePKRow{ID: 1, TenantID: 2}
	pk := row.ToPrimaryKeyMap()
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(row).Where(pk).Delete(row)
	})
	if !strings.Contains(sql, "tenant_id") {
		t.Fatalf("DELETE SQL should use composite pk, got: %s", sql)
	}
}
