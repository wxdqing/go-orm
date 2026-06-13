package sql

import (
	"github.com/wxdqing/go-orm/orm"
	"gorm.io/gorm"
)

// gormSaveRecord 持久化一行。PostgreSQL 上 GORM 默认 Save 对复合主键生成的 ON CONFLICT
// 常与 AutoMigrate 实际约束不一致（42P10），对 len(pk)>1 时改用按主键存在性分支的 Create/Updates。
func gormSaveRecord(sess *gorm.DB, dbObj interface{}) error {
	pk, ok := dbObj.(orm.PkProvider)
	if !ok {
		return sess.Save(dbObj).Error
	}
	pkMap := pk.ToPrimaryKeyMap()
	if len(pkMap) == 0 {
		return orm.ErrNoPrimaryKeySpecified
	}
	if sess.Dialector.Name() != "postgres" {
		return sess.Save(dbObj).Error
	}
	var count int64
	if err := sess.Model(dbObj).Where(pkMap).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return sess.Create(dbObj).Error
	}
	return sess.Model(dbObj).Where(pkMap).Select("*").Updates(dbObj).Error
}
