package sql

import (
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type compositePKSaveRow struct {
	ID       int64 `gorm:"column:id;primaryKey"`
	TenantID int64 `gorm:"column:tenant_id;primaryKey"`
	Name     string
}

func (compositePKSaveRow) TableName() string { return "composite_save_tbl" }

func (r *compositePKSaveRow) PrimaryKey() []any {
	return []any{r.ID, r.TenantID}
}

func (r *compositePKSaveRow) PrimaryKeyNames() []string {
	return []string{"id", "tenant_id"}
}

func (r *compositePKSaveRow) ToPrimaryKeyMap() map[string]any {
	return map[string]any{"id": r.ID, "tenant_id": r.TenantID}
}

func (r *compositePKSaveRow) ToPrimaryKeyStruct() any { return r }

// TestGormSave_PostgresCompositePK 断言 PostgreSQL 复合主键 Save 不走 ON CONFLICT upsert（T-CR-004）。
func TestGormSave_PostgresCompositePK(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 user=gorm password=gorm dbname=gorm port=5432 sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	row := &compositePKSaveRow{ID: 1, TenantID: 2, Name: "x"}
	insertSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Create(row)
	})
	updateSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(row).Where(row.ToPrimaryKeyMap()).Select("*").Updates(row)
	})
	for _, sql := range []string{insertSQL, updateSQL} {
		if strings.Contains(sql, "ON CONFLICT") {
			t.Fatalf("composite pk save should not use ON CONFLICT, got: %s", sql)
		}
	}
	if !strings.Contains(insertSQL, "INSERT") {
		t.Fatalf("expected INSERT, got: %s", insertSQL)
	}
	if !strings.Contains(updateSQL, "UPDATE") {
		t.Fatalf("expected UPDATE, got: %s", updateSQL)
	}
}
