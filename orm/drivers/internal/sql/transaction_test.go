package sql

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"gorm.io/gorm"
)

func TestRunInTxRawExecRollsBack(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tx.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE tx_records (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}

	driver := &gormBase{DB: db}
	wantErr := errors.New("rollback")
	err = driver.RunInTx(context.Background(), func(ctx context.Context) error {
		if _, err := driver.Exec(ctx, "INSERT INTO tx_records (id) VALUES (?)", 1); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("RunInTx() error = %v, want %v", err, wantErr)
	}

	var count int64
	if err := db.Table("tx_records").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rows after rollback = %d, want 0", count)
	}
}

func TestFinishInitClosesPrimaryOnValidationFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "init.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	driver := &gormBase{DB: db}
	opts := &driverapi.Options{
		Type: driverapi.TypeMySQL,
		Conf: &orm.Conf{Mysql: orm.MysqlConf{Shard: orm.SQLShardConf{
			Mode: orm.ShardModeTable,
		}}},
	}

	if err := driver.finishInit(opts, driverapi.TypeMySQL); err == nil {
		t.Fatal("finishInit() error = nil, want shard validation error")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.PingContext(context.Background()); err == nil {
		t.Fatal("primary database remains open after failed initialization")
	}
}
