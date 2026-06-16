package orm

import (
	"context"

	"gorm.io/gorm"
)

type txDBKey struct{}

// ContextWithTxDB 在 context 中绑定 GORM 事务会话，供 RunInTx 回调内 Exec/Query 使用。
func ContextWithTxDB(ctx context.Context, db *gorm.DB) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, txDBKey{}, db)
}

// TxDBFromContext 返回事务会话；无事务时回退到 fallback（并附加 ctx）。
func TxDBFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if ctx != nil {
		if v := ctx.Value(txDBKey{}); v != nil {
			if db, ok := v.(*gorm.DB); ok && db != nil {
				return db
			}
		}
	}
	if fallback == nil {
		return nil
	}
	return fallback.WithContext(ctx)
}
