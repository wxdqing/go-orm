package sql

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	logger "gitee.com/wxdqing/logger.git"
	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// gormBase 承载基于 GORM 的 SQL 驱动通用 CRUD 逻辑（MySQL / PostgreSQL）。
type gormBase struct {
	DB    *gorm.DB
	shard gormShardRouter
}

func (g *gormBase) GormDB() *gorm.DB {
	return g.DB
}

func dbRecordTableName(dbObj proto.Message) string {
	if t, ok := dbObj.(interface{ TableName() string }); ok {
		return t.TableName()
	}
	return string(dbObj.ProtoReflect().Descriptor().Name())
}

func (g *gormBase) opDB(value proto.Message, dbObj proto.Message) (*gorm.DB, error) {
	return g.shard.session(value, dbRecordTableName(dbObj))
}

func (g *gormBase) Save(ctx context.Context, tb proto.Message) error {
	meta := meta.GetMetaByValue(tb)
	if meta == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := meta.NewDbRecordFunc()
	err := codec.EncodeRecord(ctx, dbObj, tb)
	if err != nil {
		logger.Errorf("gorm encode record error,err is [%v]", err)
		return err
	}
	logger.Debugf("GormDriver.Save execute: value [%T] [%v],db obj is [%T]", tb, tb, dbObj)

	sess, err := g.opDB(tb, dbObj)
	if err != nil {
		return err
	}
	vp, isVp := dbObj.(orm.VersionProvider)
	if !isVp {
		return gormSaveRecord(sess, dbObj)
	}
	err = sess.Transaction(func(tx *gorm.DB) error {
		pkMap := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
		if len(pkMap) == 0 {
			return orm.ErrNoPrimaryKeySpecified
		}

		currentDbObj := meta.NewDbRecordFunc()
		if err := tx.Model(currentDbObj).Where(pkMap).First(currentDbObj).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				vp.SetVersion(vp.GetVersion() + 1)
				return gormSaveRecord(tx, dbObj)
			}
			return err
		}

		currentVersion := currentDbObj.(orm.VersionProvider).GetVersion()
		requestVersion := vp.GetVersion()

		if currentVersion != requestVersion {
			logger.Errorf("save version mismatch, expected: %d, actual: %d, msg(%T):%v,", requestVersion, currentVersion, tb, pkMap)
			return orm.ErrVersionMismatched
		}

		vp.SetVersion(requestVersion + 1)
		return gormSaveRecord(tx, dbObj)
	})

	if err == nil {
		tb.(orm.VersionProvider).SetVersion(vp.GetVersion())
	}
	return err
}

func (g *gormBase) Find(ctx context.Context, cond proto.Message) ([]proto.Message, error) {
	meta := meta.GetMetaByValue(cond)
	if meta == nil {
		return nil, orm.ErrNotTableRecord
	}
	dbObj := meta.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, cond); err != nil {
		logger.Errorf("gorm encode record error,err is [%v]", err)
		return nil, err
	}
	idx, err := findIndex(ctx, dbObj.(orm.IndexProvider))
	if err != nil {
		return nil, err
	}

	sess, sessErr := g.opDB(cond, dbObj)
	if sessErr != nil {
		return nil, sessErr
	}
	dbSlice := createSliceOfIdent(dbObj)
	if err := sess.Model(dbObj).Find(dbSlice, idx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, orm.ErrRecordNotFound
		}
		logger.Errorf("gorm find records error,err is [%v]", err)
		return nil, err
	}
	ret, err := decodeGormSlice(ctx, meta, dbSlice)
	if err != nil {
		return nil, err
	}
	logger.Debugf("GormDriver.Find execute: cond [%T] [%v], count [%d]", cond, cond, len(ret))
	return ret, nil
}

func findIndex(ctx context.Context, provider orm.IndexProvider) (map[string]any, error) {
	idx := provider.ToIndexMap()
	if len(idx) == 0 && !orm.FullScanAllowed(ctx) {
		return nil, orm.ErrNoIndexSpecified
	}
	return idx, nil
}

func (g *gormBase) Get(ctx context.Context, record proto.Message) (err error) {
	meta := meta.GetMetaByValue(record)
	if meta == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := meta.NewDbRecordFunc()
	err = codec.EncodeRecord(ctx, dbObj, record)
	if err != nil {
		logger.Errorf("gorm encode record error,err is [%v]", err)
		return
	}
	dest := meta.NewDbRecordFunc()
	defer func() {
		if err == nil {
			if decErr := codec.DecodeRecord(ctx, dest, record); decErr != nil {
				logger.Errorf("gorm decode record error,err is [%v]", decErr)
				err = decErr
			}
		} else if !errors.Is(err, orm.ErrRecordNotFound) {
			if decErr := codec.DecodeRecord(ctx, dbObj, record); decErr != nil {
				logger.Errorf("gorm decode record error,err is [%v]", decErr)
			}
		}
	}()

	ids := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	if len(ids) == 0 {
		err = orm.ErrNoPrimaryKeySpecified
		return
	}
	sess, sessErr := g.opDB(record, dbObj)
	if sessErr != nil {
		err = sessErr
		return
	}
	if err = sess.Model(dbObj).
		First(dest, ids).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = orm.ErrRecordNotFound
			return
		}
		logger.Errorf("gorm get record error,err is [%v]", err)
		return
	}
	logger.Debugf("GormDriver.Get execute: value [%T] [%v],db obj is [%T]", record, record, dbObj)
	return nil
}

func decodeGormSlice(ctx context.Context, meta *meta.TableMetaData, dbSlice any) ([]proto.Message, error) {
	rv := reflect.ValueOf(dbSlice)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("gorm slice has unexpected kind %s", rv.Kind())
	}
	ret := make([]proto.Message, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i)
		var row proto.Message
		if msg, ok := elem.Interface().(proto.Message); ok {
			row = msg
		} else if elem.CanAddr() {
			msg, ok := elem.Addr().Interface().(proto.Message)
			if !ok {
				return nil, orm.ErrNotTableRecord
			}
			row = msg
		} else {
			return nil, orm.ErrNotTableRecord
		}
		newVal := meta.NewValueFunc()
		if err := codec.DecodeRecord(ctx, row, newVal); err != nil {
			logger.Errorf("gorm decode record error,err is [%v]", err)
			return nil, err
		}
		ret = append(ret, newVal)
	}
	return ret, nil
}

func (g *gormBase) Delete(ctx context.Context, tb proto.Message) error {
	meta := meta.GetMetaByValue(tb)
	if meta == nil {
		return orm.ErrNotTableRecord
	}
	dbObj := meta.NewDbRecordFunc()
	err := codec.EncodeRecord(ctx, dbObj, tb)
	if err != nil {
		logger.Errorf("gorm encode record error,err is [%v]", err)
		return err
	}
	logger.Debugf("GormDriver.Delete execute: value [%T] [%v],db obj is [%T]", tb, tb, dbObj)
	sess, sessErr := g.opDB(tb, dbObj)
	if sessErr != nil {
		return sessErr
	}
	pk := dbObj.(orm.PkProvider).ToPrimaryKeyMap()
	if len(pk) == 0 {
		return orm.ErrNoPrimaryKeySpecified
	}
	return sess.Model(dbObj).Where(pk).Delete(dbObj).Error
}

func (g *gormBase) CloseDB(ctx context.Context) error {
	var firstErr error
	for _, sdb := range g.shard.shardDBs {
		if err := closeGormDB(sdb); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	g.shard.shardDBs = nil
	if err := closeGormDB(g.DB); err != nil && firstErr == nil {
		firstErr = err
	}
	g.DB = nil
	return firstErr
}

func (g *gormBase) finishInit(o *driverapi.Options, driver driverapi.Type) error {
	shardConf := sqlShardConf(o)
	shardConf.Normalize()
	if err := shardConf.Validate(); err != nil {
		return err
	}
	shardDBs, err := openDatabaseShardDBs(driver, o.Conf.Mysql, o.Conf.Pgsql, shardConf)
	if err != nil {
		return err
	}
	g.shard = newGormShardRouter(g.DB, shardDBs, shardConf)

	if err := applyDBResolver(g.DB, o); err != nil {
		return fmt.Errorf("gorm db-resolver: %w", err)
	}
	if err := applyPoolToDB(g.DB, o); err != nil {
		return fmt.Errorf("gorm primary pool: %w", err)
	}
	for _, sdb := range shardDBs {
		if err := applyPoolToDB(sdb, o); err != nil {
			return fmt.Errorf("gorm shard pool: %w", err)
		}
	}

	startup := gormStartup(o)
	tableOptions := startup.TableOptions
	dbs := []*gorm.DB{g.DB}
	if g.shard.databaseMode() {
		dbs = append(dbs, g.shard.shardDBs...)
	}
	for _, registry := range o.Tables {
		baseTable := dbRecordTableName(registry)
		rule := shardConf.TableRule(baseTable)
		for _, tname := range shardPhysicalTables(baseTable, rule) {
			for _, sdb := range dbs {
				sess := sdb
				if tableOptions != "" {
					sess = sess.Set("gorm:table_options", tableOptions)
				}
				if tname != baseTable {
					sess = sess.Table(tname)
				}
				if err := sess.AutoMigrate(registry); err != nil {
					return fmt.Errorf("automigrate table %s: %w", tname, err)
				}
			}
		}
	}
	return applyExtraMigrationsForTables(dbs, o.Tables)
}

func applyExtraMigrationsForTables(dbs []*gorm.DB, tables []proto.Message) error {
	type migrationProvider interface {
		ExtraMigrations() []string
	}
	seen := make(map[string]struct{})
	for _, registry := range tables {
		provider, ok := registry.(migrationProvider)
		if !ok {
			continue
		}
		for _, stmt := range provider.ExtraMigrations() {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, ok := seen[stmt]; ok {
				continue
			}
			seen[stmt] = struct{}{}
			for _, sdb := range dbs {
				if sdb == nil {
					continue
				}
				if err := sdb.Exec(stmt).Error; err != nil {
					return fmt.Errorf("extra migration: %w", err)
				}
			}
		}
	}
	return nil
}

func initTable(o *driverapi.Options) []any {
	tables := make([]any, 0, len(o.Tables))
	for _, registry := range o.Tables {
		tables = append(tables, registry)
	}
	return tables
}

func createSliceOfIdent(elem any) any {
	if elem == nil {
		logger.Errorf("createSliceOfIdent: elem is nil")
		return nil
	}
	elemType := reflect.TypeOf(elem)
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	sliceType := reflect.SliceOf(reflect.PtrTo(elemType))
	slicePtr := reflect.New(sliceType)
	slicePtr.Elem().Set(reflect.MakeSlice(sliceType, 0, 20))
	return slicePtr.Interface()
}
