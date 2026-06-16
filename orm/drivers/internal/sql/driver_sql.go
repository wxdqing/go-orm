package sql

import (
	"context"
	"database/sql"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/driverapi"
	"github.com/wxdqing/go-orm/orm/drivers/internal/codec"
	"github.com/wxdqing/go-orm/orm/drivers/internal/meta"
	logger "gitee.com/wxdqing/logger.git"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type errRow struct{ err error }

func (e errRow) Scan(dest ...any) error { return e.err }

func (g *gormBase) sqlConn(ctx context.Context) (*sql.DB, error) {
	if g.DB == nil {
		return nil, orm.ErrDbDriverNotInit
	}
	sess := orm.TxDBFromContext(codec.EnsureCtx(ctx), g.DB)
	return sess.DB()
}

func (g *gormBase) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	conn, err := g.sqlConn(ctx)
	if err != nil {
		return nil, err
	}
	return conn.ExecContext(codec.EnsureCtx(ctx), query, args...)
}

func (g *gormBase) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	conn, err := g.sqlConn(ctx)
	if err != nil {
		return nil, err
	}
	return conn.QueryContext(codec.EnsureCtx(ctx), query, args...)
}

func (g *gormBase) QueryRow(ctx context.Context, query string, args ...any) driverapi.Row {
	conn, err := g.sqlConn(ctx)
	if err != nil {
		return errRow{err: err}
	}
	return conn.QueryRowContext(codec.EnsureCtx(ctx), query, args...)
}

func (g *gormBase) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if g.DB == nil {
		return orm.ErrDbDriverNotInit
	}
	return g.DB.WithContext(codec.EnsureCtx(ctx)).Transaction(func(tx *gorm.DB) error {
		return fn(orm.ContextWithTxDB(ctx, tx))
	})
}

func (g *gormBase) Ping(ctx context.Context) error {
	conn, err := g.sqlConn(ctx)
	if err != nil {
		return err
	}
	return conn.PingContext(codec.EnsureCtx(ctx))
}

func (g *gormBase) Count(ctx context.Context, cond proto.Message) (int64, error) {
	tm := meta.GetMetaByValue(cond)
	if tm == nil {
		return 0, orm.ErrNotTableRecord
	}
	dbObj := tm.NewDbRecordFunc()
	if err := codec.EncodeRecord(ctx, dbObj, cond); err != nil {
		logger.Errorf("gorm encode record error,err is [%v]", err)
		return 0, err
	}
	var idx map[string]any
	if ip, ok := dbObj.(orm.IndexProvider); ok {
		idx = ip.ToIndexMap()
	}
	sess, sessErr := g.opDB(cond, dbObj)
	if sessErr != nil {
		return 0, sessErr
	}
	sess = sess.WithContext(codec.EnsureCtx(ctx)).Model(dbObj)
	if len(idx) > 0 {
		sess = sess.Where(idx)
	}
	var count int64
	if err := sess.Count(&count).Error; err != nil {
		logger.Errorf("gorm count record error,err is [%v]", err)
		return 0, err
	}
	return count, nil
}
